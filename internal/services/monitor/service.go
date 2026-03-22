package monitor

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/fresp/StatusForge/internal/models"
	"github.com/fresp/StatusForge/internal/repository"
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type Service struct {
	repo repository.MonitorRepository
}

type MonitorUpsertInput struct {
	Name            string
	Type            models.MonitorType
	Target          string
	Monitoring      models.MonitorConfig
	SSLThresholds   []int
	IntervalSeconds int
	TimeoutSeconds  int
	ComponentID     string
	SubComponentID  string
}

func NewService(repo repository.MonitorRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) ([]models.Monitor, error) {
	return s.repo.List(ctx)
}

func (s *Service) Create(ctx context.Context, input MonitorUpsertInput) (models.Monitor, error) {
	monitor, err := buildMonitor(input)
	if err != nil {
		return models.Monitor{}, err
	}

	if err := s.repo.Insert(ctx, monitor); err != nil {
		return models.Monitor{}, err
	}

	return monitor, nil
}

func (s *Service) Update(ctx context.Context, id primitive.ObjectID, input MonitorUpsertInput) (bool, error) {
	monitor, err := buildMonitor(input)
	if err != nil {
		return false, err
	}

	return s.repo.Update(ctx, id, monitor)
}

func SanitizeSSLThresholds(thresholds []int) []int {
	if len(thresholds) == 0 {
		return []int{30, 14, 7}
	}

	valid := make([]int, 0, len(thresholds))
	seen := map[int]bool{}
	for _, threshold := range thresholds {
		if threshold <= 0 {
			continue
		}
		if seen[threshold] {
			continue
		}
		seen[threshold] = true
		valid = append(valid, threshold)
	}

	if len(valid) == 0 {
		return []int{30, 14, 7}
	}

	for i := 0; i < len(valid)-1; i++ {
		for j := i + 1; j < len(valid); j++ {
			if valid[i] < valid[j] {
				valid[i], valid[j] = valid[j], valid[i]
			}
		}
	}

	return valid
}

func ValidateAdvancedOptions(monitorType models.MonitorType, target string, advanced models.MonitorAdvancedOptions) error {
	if advanced.IgnoreTLSError && !supportsIgnoreTLSError(monitorType) {
		return &ValidationError{Message: "monitoring.advanced.ignore_tls_error is only supported for http monitors"}
	}

	if advanced.CertExpiry && !supportsCertExpiry(monitorType) {
		return &ValidationError{Message: "monitoring.advanced.cert_expiry is only supported for http and ssl monitors"}
	}

	if advanced.DomainExpiry && !supportsDomainExpiry(monitorType) {
		return &ValidationError{Message: "monitoring.advanced.domain_expiry is only supported for http and ssl monitors"}
	}

	if advanced.IgnoreTLSError && advanced.CertExpiry {
		return &ValidationError{Message: "monitoring.advanced.ignore_tls_error cannot be enabled together with monitoring.advanced.cert_expiry"}
	}

	if monitorType == models.MonitorHTTP && (advanced.IgnoreTLSError || advanced.CertExpiry) {
		u, err := url.Parse(target)
		if err != nil || !strings.EqualFold(u.Scheme, "https") {
			return &ValidationError{Message: "http monitor target must use https when cert_expiry or ignore_tls_error is enabled"}
		}
	}

	if advanced.DomainExpiry {
		if _, err := extractDomain(target, monitorType); err != nil {
			return &ValidationError{Message: fmt.Sprintf("invalid domain target for domain_expiry: %v", err)}
		}
	}

	return nil
}

func buildMonitor(input MonitorUpsertInput) (models.Monitor, error) {
	if input.ComponentID == "" && input.SubComponentID == "" {
		return models.Monitor{}, &ValidationError{Message: "must specify componentId or subComponentId"}
	}

	if input.IntervalSeconds == 0 {
		input.IntervalSeconds = 60
	}

	if input.TimeoutSeconds == 0 {
		input.TimeoutSeconds = 30
	}

	if err := ValidateAdvancedOptions(input.Type, input.Target, input.Monitoring.Advanced); err != nil {
		return models.Monitor{}, err
	}

	var compID primitive.ObjectID
	var subCompID primitive.ObjectID

	if input.SubComponentID != "" && input.SubComponentID != "000000000000000000000000" {
		oid, err := primitive.ObjectIDFromHex(input.SubComponentID)
		if err != nil {
			return models.Monitor{}, &ValidationError{Message: "invalid subComponentId"}
		}
		subCompID = oid
	} else if input.ComponentID != "" && input.ComponentID != "000000000000000000000000" {
		oid, err := primitive.ObjectIDFromHex(input.ComponentID)
		if err != nil {
			return models.Monitor{}, &ValidationError{Message: "invalid componentId"}
		}
		compID = oid
	}

	monitor := models.Monitor{
		ID:              primitive.NewObjectID(),
		Name:            input.Name,
		Type:            input.Type,
		Target:          input.Target,
		Monitoring:      input.Monitoring,
		SSLThresholds:   SanitizeSSLThresholds(input.SSLThresholds),
		IntervalSeconds: input.IntervalSeconds,
		TimeoutSeconds:  input.TimeoutSeconds,
		ComponentID:     compID,
		SubComponentID:  subCompID,
		CreatedAt:       time.Now(),
	}

	return monitor, nil
}

func supportsDomainExpiry(monitorType models.MonitorType) bool {
	return monitorType == models.MonitorHTTP || monitorType == models.MonitorSSL
}

func supportsCertExpiry(monitorType models.MonitorType) bool {
	return monitorType == models.MonitorHTTP || monitorType == models.MonitorSSL
}

func supportsIgnoreTLSError(monitorType models.MonitorType) bool {
	return monitorType == models.MonitorHTTP
}

func extractDomain(target string, monitorType models.MonitorType) (string, error) {
	if monitorType == models.MonitorHTTP {
		u, err := url.Parse(target)
		if err != nil {
			return "", err
		}
		host := u.Hostname()
		if host == "" {
			return "", fmt.Errorf("target has no hostname")
		}
		if net.ParseIP(host) != nil {
			return "", fmt.Errorf("domain_expiry does not support IP targets")
		}
		return host, nil
	}

	host, _, err := net.SplitHostPort(target)
	if err != nil {
		host = target
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("empty host")
	}
	if net.ParseIP(host) != nil {
		return "", fmt.Errorf("domain_expiry does not support IP targets")
	}

	return host, nil
}
