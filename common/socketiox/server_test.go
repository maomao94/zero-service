package socketiox

import (
	"fmt"
	"testing"

	"zero-service/common/authctx"
)

func TestBuildRoomsPageResSortsAndPaginates(t *testing.T) {
	rooms := []string{"room:c", "room:a", "room:b"}

	got := buildRoomsPageRes(rooms, 1, 2)

	if got.Total != 3 {
		t.Fatalf("total = %d, want 3", got.Total)
	}
	if got.Page != 1 || got.PageSize != 2 || got.TotalPages != 2 {
		t.Fatalf("page fields = page %d pageSize %d totalPages %d, want 1/2/2", got.Page, got.PageSize, got.TotalPages)
	}
	wantRooms := []string{"room:a", "room:b"}
	if len(got.Rooms) != len(wantRooms) {
		t.Fatalf("rooms len = %d, want %d", len(got.Rooms), len(wantRooms))
	}
	for i, want := range wantRooms {
		if got.Rooms[i] != want {
			t.Fatalf("rooms[%d] = %q, want %q", i, got.Rooms[i], want)
		}
	}
}

func TestBuildRoomsPageResNormalizesDefaults(t *testing.T) {
	rooms := []string{"room:a"}

	got := buildRoomsPageRes(rooms, 0, 0)

	if got.Page != defaultRoomsPage {
		t.Fatalf("page = %d, want %d", got.Page, defaultRoomsPage)
	}
	if got.PageSize != defaultRoomsPageSize {
		t.Fatalf("pageSize = %d, want %d", got.PageSize, defaultRoomsPageSize)
	}
	if got.Total != 1 || got.TotalPages != 1 || len(got.Rooms) != 1 {
		t.Fatalf("result = total %d totalPages %d rooms %d, want 1/1/1", got.Total, got.TotalPages, len(got.Rooms))
	}
}

func TestBuildRoomsPageResCapsPageSize(t *testing.T) {
	rooms := make([]string, maxRoomsPageSize+1)
	for i := range rooms {
		rooms[i] = fmt.Sprintf("room:%03d", i)
	}

	got := buildRoomsPageRes(rooms, 1, maxRoomsPageSize+100)

	if got.PageSize != maxRoomsPageSize {
		t.Fatalf("pageSize = %d, want %d", got.PageSize, maxRoomsPageSize)
	}
	if len(got.Rooms) != maxRoomsPageSize {
		t.Fatalf("rooms len = %d, want %d", len(got.Rooms), maxRoomsPageSize)
	}
}

func TestBuildRoomsPageResOutOfRangePage(t *testing.T) {
	rooms := []string{"room:a"}

	got := buildRoomsPageRes(rooms, 2, 1)

	if got.Total != 1 || got.Page != 2 || got.PageSize != 1 || got.TotalPages != 1 {
		t.Fatalf("page result = total %d page %d pageSize %d totalPages %d, want 1/2/1/1", got.Total, got.Page, got.PageSize, got.TotalPages)
	}
	if len(got.Rooms) != 0 {
		t.Fatalf("rooms len = %d, want 0", len(got.Rooms))
	}
}

func TestVisibleSessionRoomsFiltersSocketIdRoom(t *testing.T) {
	got := visibleSessionRooms([]string{"socket-1", "room:a", "room:b"}, "socket-1")

	wantRooms := []string{"room:a", "room:b"}
	if len(got) != len(wantRooms) {
		t.Fatalf("rooms len = %d, want %d", len(got), len(wantRooms))
	}
	for i, want := range wantRooms {
		if got[i] != want {
			t.Fatalf("rooms[%d] = %q, want %q", i, got[i], want)
		}
	}
}

func TestSessionSetMetadataAuthTypeImmutable(t *testing.T) {
	session := &Session{socketId: "sess-1", metadata: make(map[string]string)}

	session.SetMetadata(authctx.CtxAuthTypeKey, "device")
	session.SetMetadata(authctx.CtxAuthTypeKey, "user")

	if got := session.GetMetadata(authctx.CtxAuthTypeKey); got != "device" {
		t.Fatalf("auth-type = %v, want device", got)
	}
}

func TestSessionSetMetadataOtherKeysOverwritable(t *testing.T) {
	session := &Session{socketId: "sess-1", metadata: make(map[string]string)}

	session.SetMetadata("userId", "u1")
	session.SetMetadata("userId", "u2")

	if got := session.GetMetadata("userId"); got != "u2" {
		t.Fatalf("userId = %v, want u2", got)
	}
}

func TestSessionSetMetadataAuthTypeInvalidValuesDoNotLock(t *testing.T) {
	session := &Session{socketId: "sess-1", metadata: make(map[string]string)}

	session.SetMetadata(authctx.CtxAuthTypeKey, "")
	session.SetMetadata(authctx.CtxAuthTypeKey, 1)
	session.SetMetadata(authctx.CtxAuthTypeKey, "user")

	if got := session.GetMetadata(authctx.CtxAuthTypeKey); got != "user" {
		t.Fatalf("auth-type = %v, want user", got)
	}
}

func TestApplyTokenClaimsDefaultAliases(t *testing.T) {
	session := &Session{socketId: "sess-1", metadata: make(map[string]string)}
	claims := map[string]any{
		"user_id":   "u1",
		"user_name": "alice",
		"dept_code": "dept-1",
		"auth-type": "user",
	}

	applyTokenClaims(session, claims, nil)

	if got := session.GetMetadata(authctx.CtxUserIdKey); got != "u1" {
		t.Fatalf("user-id = %v, want u1", got)
	}
	if got := session.GetMetadata(authctx.CtxUserNameKey); got != "alice" {
		t.Fatalf("user-name = %v, want alice", got)
	}
	if got := session.GetMetadata(authctx.CtxDeptCodeKey); got != "dept-1" {
		t.Fatalf("dept-code = %v, want dept-1", got)
	}
	if got := session.GetMetadata(authctx.CtxAuthTypeKey); got != "user" {
		t.Fatalf("auth-type = %v, want user", got)
	}
}

func TestApplyTokenClaimsCanonicalWinsOverAlias(t *testing.T) {
	session := &Session{socketId: "sess-1", metadata: make(map[string]string)}
	claims := map[string]any{
		"user-id": "canonical",
		"user_id": "snake",
		"uid":     "short",
	}

	applyTokenClaims(session, claims, nil)

	if got := session.GetMetadata(authctx.CtxUserIdKey); got != "canonical" {
		t.Fatalf("user-id = %v, want canonical", got)
	}
}

func TestApplyTokenClaimsNumericAndExtraKeys(t *testing.T) {
	session := &Session{socketId: "sess-1", metadata: make(map[string]string)}
	claims := map[string]any{
		"userId":  float64(42),
		"dept_id": "did-1",
	}

	applyTokenClaims(session, claims, []string{"dept_id", "missing"})

	if got := session.GetMetadata(authctx.CtxUserIdKey); got != "42" {
		t.Fatalf("user-id = %v, want 42", got)
	}
	if got := session.GetMetadata("dept_id"); got != "did-1" {
		t.Fatalf("dept_id = %v, want did-1", got)
	}
	if got := session.GetMetadata("missing"); got != "" {
		t.Fatalf("missing = %v, want empty", got)
	}
}

func TestEnsureAuthTypeFallback(t *testing.T) {
	// 无 auth-type → 默认 user
	noTypeSession := &Session{socketId: "sess-1", metadata: map[string]string{
		authctx.CtxUserIdKey: "u1",
	}}
	ensureAuthType(noTypeSession)
	if got := noTypeSession.GetMetadata(authctx.CtxAuthTypeKey); got != "user" {
		t.Fatalf("auth-type = %v, want user", got)
	}

	// 已有 auth-type 不可覆盖
	typedSession := &Session{socketId: "sess-2", metadata: map[string]string{
		authctx.CtxAuthTypeKey: "device",
	}}
	ensureAuthType(typedSession)
	if got := typedSession.GetMetadata(authctx.CtxAuthTypeKey); got != "device" {
		t.Fatalf("auth-type = %v, want device", got)
	}
}

func TestGetSessionByKeyAliasNormalization(t *testing.T) {
	srv := &Server{sessions: map[string]*Session{
		"sess-1": {socketId: "sess-1", metadata: map[string]string{
			authctx.CtxUserIdKey: "u1",
		}},
		"sess-2": {socketId: "sess-2", metadata: map[string]string{
			authctx.CtxUserIdKey: "u2",
			"dept_id":            "did-1",
		}},
	}}

	for _, alias := range []string{authctx.CtxUserIdKey, "userId", "user_id", "uid"} {
		sessions, ok := srv.GetSessionByKey(alias, "u2")
		if !ok || len(sessions) != 1 || sessions[0].ID() != "sess-2" {
			t.Fatalf("GetSessionByKey(%q, u2) = %v, ok=%t, want sess-2", alias, sessions, ok)
		}
	}

	// 未知键按原名比对
	sessions, ok := srv.GetSessionByKey("dept_id", "did-1")
	if !ok || len(sessions) != 1 || sessions[0].ID() != "sess-2" {
		t.Fatalf("GetSessionByKey(dept_id) = %v, ok=%t, want sess-2", sessions, ok)
	}
}

func TestWithTokenValidatorReturnsClaims(t *testing.T) {
	srv := &Server{}
	WithTokenValidator(func(token string) (map[string]any, bool) {
		if token != "ok" {
			return nil, false
		}
		return map[string]any{"user-id": "u1"}, true
	})(srv)

	if srv.tokenValidator == nil {
		t.Fatal("tokenValidator not set")
	}
	claims, valid := srv.tokenValidator("ok")
	if !valid || claims["user-id"] != "u1" {
		t.Fatalf("claims = %#v, valid = %t, want user-id/u1/true", claims, valid)
	}
	if _, valid := srv.tokenValidator("bad"); valid {
		t.Fatal("bad token accepted")
	}
}

func TestSessionNewCtxCarriesIdentity(t *testing.T) {
	session := &Session{
		socketId: "sess-1",
		token:    "token-1",
		metadata: map[string]string{
			authctx.CtxUserIdKey:   "u1",
			authctx.CtxUserNameKey: "alice",
			authctx.CtxDeptCodeKey: "dept-1",
			authctx.CtxAuthTypeKey: "user",
		},
	}

	ctx := session.NewCtx(EventUp)

	if got := authctx.GetAuthorization(ctx); got != "token-1" {
		t.Fatalf("authorization = %q, want token-1", got)
	}
	if got := authctx.GetUserId(ctx); got != "u1" {
		t.Fatalf("user-id = %q, want u1", got)
	}
	if got := authctx.GetUserName(ctx); got != "alice" {
		t.Fatalf("user-name = %q, want alice", got)
	}
	if got := authctx.GetDeptCode(ctx); got != "dept-1" {
		t.Fatalf("dept-code = %q, want dept-1", got)
	}
	if got := authctx.GetAuthType(ctx); got != "user" {
		t.Fatalf("auth-type = %q, want user", got)
	}
}
