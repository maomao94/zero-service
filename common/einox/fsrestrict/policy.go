package fsrestrict

// Policy 控制用户区（UserRoots）与会话区（SessionBaseDir/sessionId）下的读/写/改。
type Policy struct {
	ReadUser     bool
	WriteUser    bool
	EditUser     bool
	ReadSession  bool
	WriteSession bool
	EditSession  bool
}

// DefaultPolicy 推荐默认：用户区只读；会话区读写改。
func DefaultPolicy() Policy {
	return Policy{
		ReadUser: true, WriteUser: false, EditUser: false,
		ReadSession: true, WriteSession: true, EditSession: true,
	}
}

// PermissivePolicy 在用户/会话区内允许读、写、改（用于仅配置 chroot 根目录时的兼容行为）。
func PermissivePolicy() Policy {
	return Policy{
		ReadUser: true, WriteUser: true, EditUser: true,
		ReadSession: true, WriteSession: true, EditSession: true,
	}
}
