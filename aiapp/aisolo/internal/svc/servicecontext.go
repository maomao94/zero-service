package svc

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/components/model"
	ctool "github.com/cloudwego/eino/components/tool"
	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/aiapp/aisolo/internal/config"
	"zero-service/aiapp/aisolo/internal/modes"
	"zero-service/aiapp/aisolo/internal/session"
	"zero-service/aiapp/aisolo/internal/turn"
	"zero-service/common/einox/checkpoint"
	"zero-service/common/einox/fsrestrict"
	einoxkb "zero-service/common/einox/knowledge"
	"zero-service/common/einox/memory"
	"zero-service/common/einox/metrics"
	exinoModel "zero-service/common/einox/model"
	"zero-service/common/einox/tool"
	"zero-service/common/einox/tool/builtin"
	"zero-service/common/gormx"
)

// ServiceContext 服务上下文。
//
// 组装关系:
//
//	Config
//	 ├── DB (gormx 可选)
//	 ├── MemoryStorage   (消息历史)
//	 ├── SessionStore    (会话元数据 + 中断记录)
//	 ├── CheckPointStore (ADK Agent 中断 checkpoint)
//	 ├── ChatModel
//	 ├── ToolKit         (所有内置工具)
//	 ├── ModeRegistry    (5 个 Blueprint)
//	 ├── ModePool        (按 mode 缓存 Agent 实例)
//	 └── TurnExecutor    (统一 Ask / Resume 入口)
type ServiceContext struct {
	Config config.Config

	DB              *gormx.DB
	Messages        memory.Storage
	Sessions        session.Store
	CheckPointStore checkpoint.Store
	ChatModel       model.BaseChatModel
	Kit             *tool.Kit
	Registry        *modes.Registry
	Pool            *modes.Pool
	Executor        *turn.Executor
	Metrics         *metrics.Metrics
	Knowledge       *einoxkb.Service
	// KnowledgeInitErr 非空表示 Knowledge 启用但初始化失败，供 Health RPC 摘要。
	KnowledgeInitErr string
}

// NewServiceContext 构造并初始化。
func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))

	ctx := context.Background()
	s := &ServiceContext{Config: c, Metrics: metrics.Global()}

	s.initDB()
	s.initMemory()
	s.initSessions()
	s.initCheckPoint()
	s.initChatModel(ctx)
	s.initKnowledge(ctx)
	s.initKit()
	s.initModes(ctx)
	s.initExecutor()
	s.recoverRunningSessions(ctx)

	return s
}

// recoverRunningSessions 启动时清理残留 RUNNING 状态的会话。
// 场景: 上次进程异常退出 / 客户端 SSE 中途断开, 导致 session.Status 卡在 RUNNING,
// 下次用户再发消息会被 executor 拒绝 ("session is running")。此处统一清为 IDLE。
func (s *ServiceContext) recoverRunningSessions(ctx context.Context) {
	if s.Sessions == nil {
		return
	}
	n, err := s.Sessions.RecoverRunningSessions(ctx)
	if err != nil {
		logx.Errorf("[svc] recover running sessions: %v", err)
		return
	}
	if n > 0 {
		logx.Infof("[svc] recovered %d running sessions to idle (lease-aware for gormx/jsonl)", n)
	}
}

func (s *ServiceContext) initDB() {
	if !s.Config.DB.Enabled || s.Config.DB.DataSource == "" {
		return
	}
	db, err := gormx.OpenWithConf(gormx.Config{
		DataSource: s.Config.DB.DataSource,
		LogLevel:   s.Config.DB.LogLevel,
	})
	if err != nil {
		logx.Errorf("[svc] gormx open: %v", err)
		return
	}
	s.DB = db
	logx.Info("[svc] gormx ready")
}

func (s *ServiceContext) initMemory() {
	t := memory.Type(s.Config.Memory.Type)
	switch t {
	case memory.TypeGORMX:
		if s.DB == nil {
			logx.Error("[svc] memory=gormx but DB nil, fallback to in-memory")
			s.Messages = memory.NewMemoryStorage()
			return
		}
		st, err := memory.NewGormxStorage(s.DB)
		if err != nil {
			logx.Errorf("[svc] gormx memory: %v", err)
			s.Messages = memory.NewMemoryStorage()
			return
		}
		s.Messages = st
	case memory.TypeJSONL:
		st, err := memory.NewStorage(memory.Config{Type: memory.TypeJSONL, BaseDir: s.Config.Memory.BaseDir})
		if err != nil {
			logx.Errorf("[svc] jsonl memory: %v", err)
			s.Messages = memory.NewMemoryStorage()
			return
		}
		s.Messages = st
	default:
		s.Messages = memory.NewMemoryStorage()
	}
	logx.Infof("[svc] messages store=%s", t)
}

func (s *ServiceContext) initSessions() {
	cfg := session.Config{
		Type:                  s.Config.SessionStore.Type,
		BaseDir:               s.Config.SessionStore.BaseDir,
		NullLeaseRecoverGrace: s.Config.SessionStore.NullLeaseRecoverGrace,
	}
	st, err := session.NewStore(cfg, s.DB)
	if err != nil {
		logx.Errorf("[svc] session store: %v, fallback to memory", err)
		s.Sessions = session.NewMemoryStore()
		return
	}
	s.Sessions = st
	logx.Infof("[svc] sessions store=%s", s.Config.SessionStore.Type)
}

func (s *ServiceContext) initCheckPoint() {
	st, err := checkpoint.NewStore(
		checkpoint.Config{
			Type:    checkpoint.Type(s.Config.Checkpoint.Type),
			BaseDir: s.Config.Checkpoint.BaseDir,
		},
		s.DB,
	)
	if err != nil {
		logx.Errorf("[svc] checkpoint: %v, fallback to memory", err)
		s.CheckPointStore = checkpoint.NewMemoryStore()
		return
	}
	s.CheckPointStore = st
	logx.Infof("[svc] checkpoint=%s", s.Config.Checkpoint.Type)
}

func (s *ServiceContext) initChatModel(ctx context.Context) {
	cfg := exinoModel.Config{
		Provider: exinoModel.Provider(s.Config.Model.Provider),
		Model:    s.Config.Model.Model,
		APIKey:   s.Config.Model.APIKey,
	}
	if s.Config.Model.BaseURL != "" {
		cfg.BaseURL = s.Config.Model.BaseURL
	}
	cm, err := exinoModel.NewChatModel(ctx, cfg)
	if err != nil {
		logx.Errorf("[svc] chat model: %v", err)
		return
	}
	s.ChatModel = cm
	logx.Infof("[svc] chat model ready: %s/%s", s.Config.Model.Provider, s.Config.Model.Model)
}

func (s *ServiceContext) initKnowledge(ctx context.Context) {
	kb, err := einoxkb.NewService(s.Config.Knowledge, s.Config.Model.APIKey)
	if err != nil {
		logx.Errorf("[svc] knowledge: %v", err)
		s.KnowledgeInitErr = truncateInitErr(err.Error())
		return
	}
	s.Knowledge = kb
	if kb != nil {
		logx.Infof("[svc] knowledge ready backend=%s", s.Config.Knowledge.EffectiveBackend())
	}
	_ = ctx
}

func (s *ServiceContext) initKit() {
	s.Kit = builtin.NewDefaultKit()
	logx.Infof("[svc] tool kit loaded: %d tools", len(s.Kit.All()))
}

func (s *ServiceContext) initModes(ctx context.Context) {
	s.Registry = modes.NewRegistry()
	if s.ChatModel == nil {
		logx.Info("[svc] chat model nil, skip mode pool")
		return
	}
	skillsDir := config.EffectiveSkillsDir(s.Config.Skills)
	if s.Config.Skills.Enabled && s.Config.Skills.Dir != "" && skillsDir == "" {
		logx.Errorf("[svc] skills: enabled but directory missing or invalid (configure Skills.Dir or set Skills.Strict): %q", s.Config.Skills.Dir)
	}
	var deepFS fsrestrict.Config
	if s.Config.Agent.DeepLocalFilesystemEnabled() {
		if roots, err := config.ResolvedDeepFilesystemRoots(s.Config.Agent.Deep.FilesystemAllowedRoots); err != nil {
			logx.Errorf("[svc] deep filesystem user roots: %v", err)
		} else {
			deepFS.UserRoots = roots
			if len(roots) > 0 {
				logx.Infof("[svc] deep filesystem user roots: %v", roots)
			}
		}
		if s.Config.Agent.Deep.FilesystemSessionBaseDir != "" {
			if sb, err := config.ResolvedSessionBaseDir(s.Config.Agent.Deep.FilesystemSessionBaseDir); err != nil {
				logx.Errorf("[svc] deep filesystem session base: %v", err)
			} else {
				deepFS.SessionBaseDir = sb
				logx.Infof("[svc] deep filesystem session base: %s", sb)
			}
		}
		if s.Config.Agent.Deep.FilesystemLegacyUserRootsFullAccess &&
			s.Config.Agent.Deep.FilesystemSessionBaseDir == "" &&
			len(deepFS.UserRoots) > 0 {
			deepFS.Policy = fsrestrict.PermissivePolicy()
		} else {
			deepFS.Policy = s.Config.Agent.Deep.FilesystemPolicy.ToFSPolicy()
			if deepFS.Policy == (fsrestrict.Policy{}) {
				deepFS.Policy = fsrestrict.DefaultPolicy()
			}
		}
	}
	var kbTools []ctool.BaseTool
	if s.Knowledge != nil {
		if kt, err := einoxkb.NewSearchTool(s.Knowledge); err != nil {
			logx.Errorf("[svc] knowledge search tool: %v", err)
		} else if kt != nil {
			kbTools = []ctool.BaseTool{kt}
		}
	}

	s.Pool = modes.NewPool(s.Registry, modes.Dependencies{
		ChatModel:                 s.ChatModel,
		Kit:                       s.Kit,
		CheckPointStore:           s.CheckPointStore,
		SkillsDir:                 skillsDir,
		DeepEnableLocalFilesystem: s.Config.Agent.DeepLocalFilesystemEnabled(),
		DeepFSConfig:              deepFS,
		PlanMaxIterations:         s.Config.Agent.EffectivePlanMaxIterations(),
		DemoSurveyEcho:            s.Config.Agent.DemoSurveyEcho,
		KnowledgeTools:            kbTools,
	})
	_ = ctx
	logx.Infof("[svc] mode registry + pool ready: %d modes", len(s.Registry.List()))
}

func (s *ServiceContext) initExecutor() {
	if s.Pool == nil {
		logx.Info("[svc] pool nil, skip executor")
		return
	}
	instanceID := s.Config.EffectiveRunInstanceID()
	s.Executor = turn.New(turn.Config{
		Pool:              s.Pool,
		Registry:          s.Registry,
		Messages:          s.Messages,
		Sessions:          s.Sessions,
		Metrics:           s.Metrics,
		App:               &s.Config,
		RunInstanceID:     instanceID,
		RunLeaseTTL:       s.Config.EffectiveSessionRunLeaseTTL(),
		RunNullLeaseGrace: s.Config.EffectiveNullLeaseRecoverGrace(),
	})
	logx.Info("[svc] turn executor ready")
}

// Close 释放资源。
func (s *ServiceContext) Close() error {
	if s.Pool != nil {
		s.Pool.Cleanup()
	}
	if s.Sessions != nil {
		_ = s.Sessions.Close()
	}
	if s.Messages != nil {
		_ = s.Messages.Close()
	}
	if s.CheckPointStore != nil {
		_ = s.CheckPointStore.Close()
	}
	if s.Knowledge != nil {
		_ = s.Knowledge.Close()
	}
	return nil
}

func truncateInitErr(s string) string {
	const max = 512
	if len(s) <= max {
		return s
	}
	return strings.TrimSpace(s[:max])
}
