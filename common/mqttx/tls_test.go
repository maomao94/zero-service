package mqttx

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testCACertPEM = `-----BEGIN CERTIFICATE-----
MIIDEzCCAfugAwIBAgIUL1FMwMQiZKzgd3gqdoRchw72RTwwDQYJKoZIhvcNAQEL
BQAwGDEWMBQGA1UEAwwNbXF0dHgtdGVzdC1jYTAgFw0yNjExMTYwNjEwMTZaGA8y
MTI2MTAyMzA2MTAxNlowGDEWMBQGA1UEAwwNbXF0dHgtdGVzdC1jYTCCASIwDQYJ
KoZIhvcNAQEBBQADggEPADCCAQoCggEBAJpVeWYyfW6n8V5W0FXWUM3ljJH6Ly6Q
Dbv6IBQzLRQKBlCSSjgp9hG3FC7gjQdGyIEDwSyQWu/y396ASfH4a0G/PdKJxTGt
Xe4TzOIPxEPQeyIA3355SLhnOV6IGazc+fK5fLyM+C2Bw3xmfjEJ8n7uabNJCq/J
t4qGbxPlNpualxttYivuLN4DPsuS5y1RFFOrfcggGOu/WQIq6EKMR0s7TNr9HjdV
xPKGTpcmf2c3AyZ6mX9j4z1AnV6Vbv28YLGclW/Zu4fWK0J2mNUN9PeiGUzx9uK1
PsAZMLDXLbhqBI85QQUyPb2H5JLp0D8REq4y3qHtwnH7xc9yiBuI7dUCAwEAAaNT
MFEwHQYDVR0OBBYEFAclsBkKREogt75XxRxZBXLLx8i9MB8GA1UdIwQYMBaAFAcl
sBkKREogt75XxRxZBXLLx8i9MA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQEL
BQADggEBAAh712UL5mAOah2yfRlXktJE7kZrdec1cBU9byw5rCsUnZJ8fhqbCIbi
HL1EFFEzvhzT0vziE6RA/BpZd1T1JQIVsb+6i9zRzbZl2eATfAGS993d12+JcJt6
qAbLxT+yMMPz+tusPnzEu6vJ/isD72GerWfP/iY0hOKbfRTmjpILQ2Z+pgD0fmM/
LpqxTe/1DRA+Kii1fhrzRcJfLvpJg+sPrbznuj/GzJDBBSxD6F/qhkbFi8Ot75MQ
mxFE93QDUpCyoxJBzs9yunEkYsc0DBlvcbMELxhgnYkIt+Tz2faa6H2x5xiZMR6G
2jIpoQ8xFa18EGhkzEC5Mf9MUJ02ZCk=
-----END CERTIFICATE-----
`

func writeTestFile(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write test file %s: %v", name, err)
	}
	return path
}

func TestBuildTLSConfigReturnsNilWhenNotConfigured(t *testing.T) {
	c := &mqttClient{cfg: MqttConfig{Broker: []string{"ssl://broker:8883"}}}

	tlsCfg, err := c.buildTLSConfig()
	if err != nil {
		t.Fatalf("buildTLSConfig returned error: %v", err)
	}
	if tlsCfg != nil {
		t.Fatalf("expected nil TLS config when TLS section is absent, got %+v", tlsCfg)
	}
}

func TestBuildTLSConfigLoadsCAFile(t *testing.T) {
	caFile := writeTestFile(t, "cacert.pem", testCACertPEM)
	c := &mqttClient{cfg: MqttConfig{TLS: TLSConfig{CAFile: caFile}}}

	tlsCfg, err := c.buildTLSConfig()
	if err != nil {
		t.Fatalf("buildTLSConfig returned error: %v", err)
	}
	if tlsCfg == nil {
		t.Fatal("expected non-nil TLS config")
	}
	if tlsCfg.RootCAs == nil {
		t.Fatal("expected RootCAs to be populated from CAFile")
	}
	if tlsCfg.MinVersion != tls.VersionTLS12 {
		t.Fatalf("expected min TLS version 1.2, got %d", tlsCfg.MinVersion)
	}
}

func TestBuildTLSConfigPropagatesServerNameAndSkipVerify(t *testing.T) {
	c := &mqttClient{cfg: MqttConfig{TLS: TLSConfig{
		ServerName:         "mq.example.com",
		InsecureSkipVerify: true,
	}}}

	tlsCfg, err := c.buildTLSConfig()
	if err != nil {
		t.Fatalf("buildTLSConfig returned error: %v", err)
	}
	if tlsCfg == nil {
		t.Fatal("expected non-nil TLS config")
	}
	if tlsCfg.ServerName != "mq.example.com" {
		t.Fatalf("ServerName = %q, want mq.example.com", tlsCfg.ServerName)
	}
	if !tlsCfg.InsecureSkipVerify {
		t.Fatal("expected InsecureSkipVerify to be propagated")
	}
}

func TestBuildTLSConfigRejectsInvalidCAFile(t *testing.T) {
	cases := map[string]struct {
		caFile string
		want   string
	}{
		"missing file": {
			caFile: filepath.Join(t.TempDir(), "missing.pem"),
			want:   "read CA file failed",
		},
		"invalid pem": {
			caFile: writeTestFile(t, "invalid.pem", "not a certificate"),
			want:   "no valid certificate found",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &mqttClient{cfg: MqttConfig{TLS: TLSConfig{CAFile: tc.caFile}}}
			_, err := c.buildTLSConfig()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestBuildTLSConfigRequiresCertAndKeyTogether(t *testing.T) {
	c := &mqttClient{cfg: MqttConfig{TLS: TLSConfig{CertFile: "client.pem"}}}

	_, err := c.buildTLSConfig()
	if err == nil {
		t.Fatal("expected error when keyFile is missing")
	}
	if !strings.Contains(err.Error(), "certFile and keyFile") {
		t.Fatalf("error = %q, want cert/key pairing message", err)
	}
}
