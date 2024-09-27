package resource

import (
	"context"
	"errors"
	"strings"

	"github.com/renderinc/render-cli/pkg/client"
	"github.com/renderinc/render-cli/pkg/environment"
	"github.com/renderinc/render-cli/pkg/postgres"
	"github.com/renderinc/render-cli/pkg/project"
	"github.com/renderinc/render-cli/pkg/service"
)

type Resource interface {
	ID() string
	Name() string
	Environment() *client.Environment
	EnvironmentName() string
	Project() *client.Project
	ProjectName() string
	Type() string
}

type Service struct {
	serviceService  *service.Service
	postgresService *postgres.Service
	environmentRepo *environment.Repo
	projectRepo     *project.Repo
}

func NewResourceService(serviceService *service.Service, postgresService *postgres.Service, environmentRepo *environment.Repo, projectRepo *project.Repo) *Service {
	return &Service{
		serviceService:  serviceService,
		postgresService: postgresService,
		environmentRepo: environmentRepo,
		projectRepo:     projectRepo,
	}
}

func (rs *Service) ListResources(ctx context.Context) ([]Resource, error) {
	services, err := rs.serviceService.ListServices(ctx)
	if err != nil {
		return nil, err
	}

	postgresDBs, err := rs.postgresService.ListPostgres(ctx)
	if err != nil {
		return nil, err
	}

	var resources []Resource

	for _, svc := range services {
		resources = append(resources, svc)
	}

	for _, db := range postgresDBs {
		resources = append(resources, db)
	}

	return resources, nil
}

func (rs *Service) RestartResource(ctx context.Context, id string) error {
	if strings.HasPrefix(id, service.ServerResourceIDPrefix) {
		return rs.serviceService.RestartService(ctx, id)
	}

	if strings.HasPrefix(id, postgres.ResourceIDPrefix) {
		return rs.postgresService.RestartPostgresDatabase(ctx, id)
	}

	if strings.HasPrefix(id, service.CronjobResourceIDPrefix) {
		return errors.New("cron jobs cannot be restarted")
	}

	return errors.New("unknown resource type")
}