// Package middleware provides mTLS (mutual TLS) authentication
package middleware

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strings"

	"log/slog"

	"radgateway/internal/logger"
)

// MTLSConfig configures mutual TLS authentication
type MTLSConfig struct {
	// Enabled enables mTLS
	Enabled bool

	// CertFile is the path to the TLS certificate
	CertFile string

	// KeyFile is the path to the TLS private key
	KeyFile string

	// CAFile is the path to the CA certificate for client verification
	CAFile string

	// ClientAuth controls client certificate verification
	// NoClientCert - no client cert required
	// RequestClientCert - request but don't require
	// RequireAnyClientCert - require any valid cert
	// VerifyClientCertIfGiven - verify if provided
	// RequireAndVerifyClientCert - require and verify (full mTLS)
	ClientAuth tls.ClientAuthType

	// SkipVerify skips client cert verification (for testing only)
	SkipVerify bool

	// AllowedCNs is a list of allowed client certificate Common Names
	AllowedCNs []string

	// AllowedOUs is a list of allowed client certificate Organizational Units
	AllowedOUs []string
}

// DefaultMTLSConfig returns default mTLS configuration
func DefaultMTLSConfig() MTLSConfig {
	return MTLSConfig{
		Enabled:    false,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}
}

// LoadMTLSConfig loads mTLS configuration from environment
func LoadMTLSConfig() MTLSConfig {
	cfg := DefaultMTLSConfig()

	if getenv("RAD_TLS_ENABLED", "false") == "true" {
		cfg.Enabled = true
	}

	cfg.CertFile = getenv("RAD_TLS_CERT_FILE", "")
	cfg.KeyFile = getenv("RAD_TLS_KEY_FILE", "")
	cfg.CAFile = getenv("RAD_TLS_CA_FILE", "")

	// Parse client auth mode
	switch getenv("RAD_TLS_CLIENT_AUTH", "require") {
	case "none", "no":
		cfg.ClientAuth = tls.NoClientCert
	case "request":
		cfg.ClientAuth = tls.RequestClientCert
	case "require-any":
		cfg.ClientAuth = tls.RequireAnyClientCert
	case "verify-if-given":
		cfg.ClientAuth = tls.VerifyClientCertIfGiven
	default:
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	cfg.SkipVerify = getenv("RAD_TLS_SKIP_VERIFY", "false") == "true"

	// Parse allowed CNs
	if cnStr := getenv("RAD_TLS_ALLOWED_CN", ""); cnStr != "" {
		cfg.AllowedCNs = strings.Split(cnStr, ",")
		for i := range cfg.AllowedCNs {
			cfg.AllowedCNs[i] = strings.TrimSpace(cfg.AllowedCNs[i])
		}
	}

	// Parse allowed OUs
	if ouStr := getenv("RAD_TLS_ALLOWED_OU", ""); ouStr != "" {
		cfg.AllowedOUs = strings.Split(ouStr, ",")
		for i := range cfg.AllowedOUs {
			cfg.AllowedOUs[i] = strings.TrimSpace(cfg.AllowedOUs[i])
		}
	}

	return cfg
}

// Validate checks if the configuration is valid
func (c MTLSConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.CertFile == "" {
		return fmt.Errorf("TLS certificate file is required")
	}

	if c.KeyFile == "" {
		return fmt.Errorf("TLS key file is required")
	}

	if c.CAFile == "" && c.ClientAuth >= tls.VerifyClientCertIfGiven {
		return fmt.Errorf("CA file is required for client certificate verification")
	}

	// Check files exist
	if _, err := os.Stat(c.CertFile); err != nil {
		return fmt.Errorf("certificate file not found: %w", err)
	}

	if _, err := os.Stat(c.KeyFile); err != nil {
		return fmt.Errorf("key file not found: %w", err)
	}

	if c.CAFile != "" {
		if _, err := os.Stat(c.CAFile); err != nil {
			return fmt.Errorf("CA file not found: %w", err)
		}
	}

	return nil
}

// TLSConfig returns a tls.Config for use with http.Server
func (c MTLSConfig) TLSConfig() (*tls.Config, error) {
	if !c.Enabled {
		return nil, nil
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}

	// Load server certificate
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   c.ClientAuth,
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: true,
	}

	// Configure client certificate verification
	if c.CAFile != "" && c.ClientAuth >= tls.VerifyClientCertIfGiven {
		caCert, err := os.ReadFile(c.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		config.ClientCAs = caCertPool
	}

	if c.SkipVerify {
		config.InsecureSkipVerify = true
	}

	return config, nil
}

// MTLSMiddleware provides mTLS client certificate validation
type MTLSMiddleware struct {
	config MTLSConfig
	log    *slog.Logger
}

// NewMTLSMiddleware creates a new mTLS middleware
func NewMTLSMiddleware(config MTLSConfig) *MTLSMiddleware {
	return &MTLSMiddleware{
		config: config,
		log:    logger.WithComponent("mtls"),
	}
}

// Handler validates client certificates and extracts identity
func (m *MTLSMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Check TLS connection
		if r.TLS == nil {
			m.log.Warn("TLS connection required but not present", "path", r.URL.Path)
			http.Error(w, `{"error":{"message":"TLS connection required","code":400}}`, http.StatusBadRequest)
			return
		}
		if r.TLS.VerifiedChains == nil && m.config.ClientAuth >= tls.RequireAnyClientCert {
			m.log.Warn("client certificate required but not present", "path", r.URL.Path, "remote_addr", r.RemoteAddr)
			http.Error(w, `{"error":{"message":"client certificate required","code":401}}`, http.StatusUnauthorized)
			return
		}
		if len(r.TLS.PeerCertificates) > 0 {
			cert := r.TLS.PeerCertificates[0]
			m.log.Debug("client certificate presented",
				"subject", cert.Subject.String(),
				"issuer", cert.Issuer.String(),
				"path", r.URL.Path,
			)
			// Validate allowed CNs
			if len(m.config.AllowedCNs) > 0 {
				found := false
				for _, cn := range m.config.AllowedCNs {
					if cert.Subject.CommonName == cn {
						found = true
						break
					}
				}
				if !found {
					m.log.Warn("client certificate CN not allowed",
						"cn", cert.Subject.CommonName,
						"allowed_cns", m.config.AllowedCNs,
						"path", r.URL.Path,
					)
					http.Error(w, `{"error":{"message":"client certificate not authorized","code":403}}`, http.StatusForbidden)
					return
				}
			}
			// Validate allowed OUs
			if len(m.config.AllowedOUs) > 0 {
				found := false
				for _, ou := range m.config.AllowedOUs {
					for _, certOU := range cert.Subject.OrganizationalUnit {
						if certOU == ou {
							found = true
							break
						}
					}
					if found {
						break
					}
				}
				if !found {
					m.log.Warn("client certificate OU not allowed",
						"ous", cert.Subject.OrganizationalUnit,
						"allowed_ous", m.config.AllowedOUs,
						"path", r.URL.Path,
					)
					http.Error(w, `{"error":{"message":"client certificate OU not authorized","code":403}}`, http.StatusForbidden)
					return
				}
			}
			// Add certificate info to context
			ctx := WithClientCert(r.Context(), cert)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// ClientCertKey is the context key for client certificates
type ClientCertKey struct{}

// WithClientCert adds client certificate to context
func WithClientCert(ctx context.Context, cert *x509.Certificate) context.Context {
	return context.WithValue(ctx, ClientCertKey{}, cert)
}

// GetClientCert retrieves client certificate from context
func GetClientCert(ctx context.Context) *x509.Certificate {
	if cert, ok := ctx.Value(ClientCertKey{}).(*x509.Certificate); ok {
		return cert
	}
	return nil
}

// GetClientCertSubject returns the client certificate subject
func GetClientCertSubject(ctx context.Context) string {
	if cert := GetClientCert(ctx); cert != nil {
		return cert.Subject.String()
	}
	return ""
}

// GetClientCertCN returns the client certificate Common Name
func GetClientCertCN(ctx context.Context) string {
	if cert := GetClientCert(ctx); cert != nil {
		return cert.Subject.CommonName
	}
	return ""
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
