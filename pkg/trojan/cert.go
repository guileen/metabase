package trojan

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CertificateManager manages TLS certificates
type CertificateManager struct {
	config      *TrojanConfig
	cert        *tls.Certificate
	certMutex   sync.RWMutex
	stopChan    chan struct{}
	running     bool
	runningLock sync.RWMutex
}

// NewCertificateManager creates a new certificate manager
func NewCertificateManager(config *TrojanConfig) (*CertificateManager, error) {
	mgr := &CertificateManager{
		config:   config,
		stopChan: make(chan struct{}),
	}

	// Load existing certificate or generate new one
	if err := mgr.loadCertificate(); err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	return mgr, nil
}

// loadCertificate loads an existing certificate or generates a new one
func (cm *CertificateManager) loadCertificate() error {
	if cm.config.AutoCert {
		// Generate self-signed certificate
		return cm.generateSelfSignedCert()
	}

	// Load certificate from files
	if cm.config.CertPath != "" && cm.config.KeyPath != "" {
		return cm.loadCertificateFromFiles()
	}

	return fmt.Errorf("no certificate configuration provided")
}

// generateSelfSignedCert generates a self-signed certificate
func (cm *CertificateManager) generateSelfSignedCert() error {
	// Generate private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // 1 year

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"MetaBase Trojan"},
			CommonName:   cm.config.ServerName,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{cm.config.ServerName, "localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	// If host is not an IP address, add it to DNS names
	if ip := net.ParseIP(cm.config.Host); ip == nil {
		template.DNSNames = append(template.DNSNames, cm.config.Host)
	} else {
		template.IPAddresses = append(template.IPAddresses, ip)
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Create certificate
	cert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  privateKey,
		Leaf:        &template,
	}

	cm.certMutex.Lock()
	cm.cert = &cert
	cm.certMutex.Unlock()

	return nil
}

// loadCertificateFromFiles loads certificate and key from files
func (cm *CertificateManager) loadCertificateFromFiles() error {
	certFile, err := os.ReadFile(cm.config.CertPath)
	if err != nil {
		return fmt.Errorf("failed to read certificate file: %w", err)
	}

	keyFile, err := os.ReadFile(cm.config.KeyPath)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	cert, err := tls.X509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to load certificate and key: %w", err)
	}

	cm.certMutex.Lock()
	cm.cert = &cert
	cm.certMutex.Unlock()

	return nil
}

// GetCertificate implements the tls.Config.GetCertificate callback
func (cm *CertificateManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	cm.certMutex.RLock()
	defer cm.certMutex.RUnlock()

	if cm.cert == nil {
		return nil, fmt.Errorf("no certificate available")
	}

	return cm.cert, nil
}

// Start starts the certificate manager
func (cm *CertificateManager) Start() {
	cm.runningLock.Lock()
	defer cm.runningLock.Unlock()

	if cm.running {
		return
	}

	cm.running = true

	// Start certificate renewal if using auto-cert
	if cm.config.AutoCert {
		go cm.renewalLoop()
	}
}

// Stop stops the certificate manager
func (cm *CertificateManager) Stop() {
	cm.runningLock.Lock()
	defer cm.runningLock.Unlock()

	if !cm.running {
		return
	}

	cm.running = false
	close(cm.stopChan)
}

// renewalLoop periodically renews the certificate
func (cm *CertificateManager) renewalLoop() {
	ticker := time.NewTicker(24 * time.Hour) // Check daily
	defer ticker.Stop()

	for {
		select {
		case <-cm.stopChan:
			return
		case <-ticker.C:
			if cm.shouldRenewCertificate() {
				if err := cm.generateSelfSignedCert(); err != nil {
					// Log error but continue running
					fmt.Printf("Failed to renew certificate: %v\n", err)
				}
			}
		}
	}
}

// shouldRenewCertificate checks if the certificate needs renewal
func (cm *CertificateManager) shouldRenewCertificate() bool {
	cm.certMutex.RLock()
	defer cm.certMutex.RUnlock()

	if cm.cert == nil || cm.cert.Leaf == nil {
		return true
	}

	// Renew if certificate expires within 30 days
	return time.Until(cm.cert.Leaf.NotAfter) < 30*24*time.Hour
}

// SaveCertificate saves the current certificate to files
func (cm *CertificateManager) SaveCertificate(certPath, keyPath string) error {
	cm.certMutex.RLock()
	defer cm.certMutex.RUnlock()

	if cm.cert == nil {
		return fmt.Errorf("no certificate to save")
	}

	// Create directories if they don't exist
	if err := os.MkdirAll(filepath.Dir(certPath), 0755); err != nil {
		return fmt.Errorf("failed to create certificate directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0755); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// Save certificate
	certOut, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("failed to create certificate file: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cm.cert.Certificate[0],
	}); err != nil {
		return fmt.Errorf("failed to encode certificate: %w", err)
	}

	// Save private key
	keyBytes, err := x509.MarshalPKCS8PrivateKey(cm.cert.PrivateKey.(crypto.PrivateKey))
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	keyOut, err := os.Create(keyPath)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyOut.Close()

	if err := pem.Encode(keyOut, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	}); err != nil {
		return fmt.Errorf("failed to encode private key: %w", err)
	}

	return nil
}

// GetCertificateInfo returns information about the current certificate
func (cm *CertificateManager) GetCertificateInfo() *CertificateInfo {
	cm.certMutex.RLock()
	defer cm.certMutex.RUnlock()

	if cm.cert == nil || cm.cert.Leaf == nil {
		return &CertificateInfo{
			Available: false,
		}
	}

	info := &CertificateInfo{
		Available:    true,
		NotBefore:    cm.cert.Leaf.NotBefore,
		NotAfter:     cm.cert.Leaf.NotAfter,
		DNSNames:     cm.cert.Leaf.DNSNames,
		IPAddresses:  make([]string, len(cm.cert.Leaf.IPAddresses)),
		IsAutoCert:   cm.config.AutoCert,
		Subject:      cm.cert.Leaf.Subject.CommonName,
		Issuer:       cm.cert.Leaf.Issuer.CommonName,
		SerialNumber: cm.cert.Leaf.SerialNumber.String(),
	}

	for i, ip := range cm.cert.Leaf.IPAddresses {
		info.IPAddresses[i] = ip.String()
	}

	info.DaysUntilExpiry = int(time.Until(cm.cert.Leaf.NotAfter).Hours() / 24)

	return info
}

// CertificateInfo contains certificate information
type CertificateInfo struct {
	Available       bool      `json:"available"`
	NotBefore       time.Time `json:"not_before"`
	NotAfter        time.Time `json:"not_after"`
	DNSNames        []string  `json:"dns_names"`
	IPAddresses     []string  `json:"ip_addresses"`
	IsAutoCert      bool      `json:"is_auto_cert"`
	Subject         string    `json:"subject"`
	Issuer          string    `json:"issuer"`
	SerialNumber    string    `json:"serial_number"`
	DaysUntilExpiry int       `json:"days_until_expiry"`
}
