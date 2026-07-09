package config

import "zero-service/common/ftps"

// ToFTPSConfig maps ispagent model-sync configuration into the shared FTPS uploader config.
func (c ModelSyncFTPSConfig) ToFTPSConfig() ftps.Config {
	return ftps.Config{
		Address:            c.Address,
		Username:           c.Username,
		Password:           c.Password,
		RemoteDir:          c.RemoteDir,
		TLSMode:            ftps.TLSMode(c.TLSMode),
		InsecureSkipVerify: c.InsecureSkipVerify,
		Timeout:            c.Timeout,
		DisableEPSV:        c.DisableEPSV,
		UseTemporaryFile:   c.UseTemporaryFile,
	}
}
