package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/repository/sqlc"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool    *pgxpool.Pool
	db      sqlc.DBTX
	queries *sqlc.Queries
}

func NewPostgres(ctx context.Context, databaseURL string) (*PostgresRepository, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("DOCUMENT_DATABASE_URL is required")
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, errors.New("DOCUMENT_DATABASE_URL is invalid")
	}
	config.MaxConns = 10
	config.MinConns = 1
	config.MaxConnLifetime = 30 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	return NewPostgresRepository(pool), nil
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool, db: pool, queries: sqlc.New(pool)}
}

func (r *PostgresRepository) Close() {
	r.pool.Close()
}

func (r *PostgresRepository) CheckReady(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *PostgresRepository) WithinTx(ctx context.Context, fn func(service.ReportRepository) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin document transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txRepo := &PostgresRepository{
		pool:    r.pool,
		db:      tx,
		queries: r.queries.WithTx(tx),
	}
	if err := fn(txRepo); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit document transaction: %w", err)
	}
	return nil
}

func (r *PostgresRepository) UpsertReportType(ctx context.Context, value service.ReportType) (service.ReportType, error) {
	if value.CreatedAt.IsZero() {
		value.CreatedAt = time.Now().UTC()
	}
	if value.UpdatedAt.IsZero() {
		value.UpdatedAt = value.CreatedAt
	}
	defaultTemplateID, err := parseOptionalUUIDField(value.DefaultTemplateID, "defaultTemplateId")
	if err != nil {
		return service.ReportType{}, err
	}
	row := r.db.QueryRow(ctx, `
		INSERT INTO report_types (
			code, name, description, enabled, default_template_id, created_at, updated_at
		)
		VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, '')::uuid, $6, $7)
		ON CONFLICT (code) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			enabled = EXCLUDED.enabled,
			default_template_id = EXCLUDED.default_template_id,
			updated_at = EXCLUDED.updated_at
		RETURNING code, name, COALESCE(description, ''), enabled, COALESCE(default_template_id::text, ''), created_at, updated_at`,
		value.Code,
		value.Name,
		value.Description,
		value.Enabled,
		defaultTemplateID,
		value.CreatedAt,
		value.UpdatedAt,
	)
	return scanReportType(row)
}

func (r *PostgresRepository) ListReportTypes(ctx context.Context) ([]service.ReportType, error) {
	rows, err := r.db.Query(ctx, `
		SELECT code, name, COALESCE(description, ''), enabled, COALESCE(default_template_id::text, ''), created_at, updated_at
		FROM report_types
		WHERE enabled = true
		ORDER BY code`)
	if err != nil {
		return nil, fmt.Errorf("list report types: %w", err)
	}
	defer rows.Close()

	values := []service.ReportType{}
	for rows.Next() {
		value, err := scanReportType(rows)
		if err != nil {
			return nil, fmt.Errorf("scan report type: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate report types: %w", err)
	}
	return values, nil
}

func (r *PostgresRepository) ReportTypeExists(ctx context.Context, code string) (bool, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM report_types WHERE code = $1 AND enabled = true
		)`, code).Scan(&exists); err != nil {
		return false, fmt.Errorf("check report type exists: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) ListReportTemplates(ctx context.Context, filter service.ReportTemplateListFilter) (service.ReportTemplateListResult, error) {
	where := []string{"deleted_at IS NULL"}
	args := []any{}
	if strings.TrimSpace(filter.ReportType) != "" {
		args = append(args, strings.TrimSpace(filter.ReportType))
		where = append(where, fmt.Sprintf("report_type = $%d", len(args)))
	}
	if filter.Enabled != nil {
		args = append(args, *filter.Enabled)
		where = append(where, fmt.Sprintf("enabled = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	countQuery := "SELECT count(*) FROM report_templates WHERE " + whereSQL
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return service.ReportTemplateListResult{}, fmt.Errorf("count report templates: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	queryArgs := append(append([]any{}, args...), filter.PageSize, offset)
	query := fmt.Sprintf(`
		SELECT
			id::text, template_name, report_type, version, COALESCE(file_ref, ''),
			filename, file_size, COALESCE(description, ''), enabled, COALESCE(created_by, ''),
			created_at, updated_at, deleted_at
		FROM report_templates
		WHERE %s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d OFFSET $%d`, whereSQL, len(queryArgs)-1, len(queryArgs))
	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return service.ReportTemplateListResult{}, fmt.Errorf("list report templates: %w", err)
	}
	defer rows.Close()

	items := []service.ReportTemplate{}
	for rows.Next() {
		item, err := scanReportTemplate(rows)
		if err != nil {
			return service.ReportTemplateListResult{}, fmt.Errorf("scan report template: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return service.ReportTemplateListResult{}, fmt.Errorf("iterate report templates: %w", err)
	}
	return service.ReportTemplateListResult{
		Items: items,
		Page:  service.PageMeta{Page: filter.Page, PageSize: filter.PageSize, Total: int(total)},
	}, nil
}

func (r *PostgresRepository) CreateReportTemplate(ctx context.Context, value service.ReportTemplate, structure service.ReportTemplateStructure) (service.ReportTemplate, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO report_templates (
			id, template_name, report_type, version, file_ref, filename, file_size,
			structure_json, style_config_json, description, enabled, created_by, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, NULLIF($5, ''), $6, $7,
			$8::jsonb, $9::jsonb, NULLIF($10, ''), $11, NULLIF($12, ''), $13, $14
		)
		RETURNING
			id::text, template_name, report_type, version, COALESCE(file_ref, ''),
			filename, file_size, COALESCE(description, ''), enabled, COALESCE(created_by, ''),
			created_at, updated_at, deleted_at`,
		value.ID,
		value.TemplateName,
		value.ReportType,
		value.Version,
		value.FileRef,
		value.Filename,
		value.FileSize,
		string(structure.OutlineSchema),
		string(structure.StyleConfig),
		value.Description,
		value.Enabled,
		value.CreatedBy,
		value.CreatedAt,
		value.UpdatedAt,
	)
	created, err := scanReportTemplate(row)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ReportTemplate{}, service.NewError(service.CodeConflict, "report template already exists", err)
		}
		return service.ReportTemplate{}, fmt.Errorf("insert report template: %w", err)
	}
	return created, nil
}

func (r *PostgresRepository) FindReportTemplateByID(ctx context.Context, id string) (service.ReportTemplate, error) {
	templateID, err := parseUUID(id)
	if err != nil {
		return service.ReportTemplate{}, service.ValidationError(map[string]string{"reportTemplateId": "must be a valid UUID"})
	}
	row := r.db.QueryRow(ctx, `
		SELECT
			id::text, template_name, report_type, version, COALESCE(file_ref, ''),
			filename, file_size, COALESCE(description, ''), enabled, COALESCE(created_by, ''),
			created_at, updated_at, deleted_at
		FROM report_templates
		WHERE id = $1 AND deleted_at IS NULL`, templateID)
	template, err := scanReportTemplate(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.ReportTemplate{}, service.NewError(service.CodeNotFound, "report template not found", err)
		}
		return service.ReportTemplate{}, fmt.Errorf("find report template: %w", err)
	}
	return template, nil
}

func (r *PostgresRepository) UpdateReportTemplate(ctx context.Context, input service.UpdateReportTemplateInput) (service.ReportTemplate, error) {
	templateID, err := parseUUID(input.ID)
	if err != nil {
		return service.ReportTemplate{}, service.ValidationError(map[string]string{"reportTemplateId": "must be a valid UUID"})
	}
	templateName := ""
	if input.TemplateName != nil {
		templateName = strings.TrimSpace(*input.TemplateName)
	}
	description := ""
	if input.Description != nil {
		description = strings.TrimSpace(*input.Description)
	}
	enabled := false
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	row := r.db.QueryRow(ctx, `
		UPDATE report_templates
		SET
			template_name = CASE WHEN $2 THEN $3 ELSE template_name END,
			description = CASE WHEN $4 THEN NULLIF($5, '') ELSE description END,
			enabled = CASE WHEN $6 THEN $7 ELSE enabled END,
			updated_at = $8
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING
			id::text, template_name, report_type, version, COALESCE(file_ref, ''),
			filename, file_size, COALESCE(description, ''), enabled, COALESCE(created_by, ''),
			created_at, updated_at, deleted_at`,
		templateID,
		input.TemplateName != nil,
		templateName,
		input.Description != nil,
		description,
		input.Enabled != nil,
		enabled,
		time.Now().UTC(),
	)
	template, err := scanReportTemplate(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.ReportTemplate{}, service.NewError(service.CodeNotFound, "report template not found", err)
		}
		return service.ReportTemplate{}, fmt.Errorf("update report template: %w", err)
	}
	return template, nil
}

func (r *PostgresRepository) DeleteReportTemplate(ctx context.Context, id string, deletedAt time.Time) error {
	templateID, err := parseUUID(id)
	if err != nil {
		return service.ValidationError(map[string]string{"reportTemplateId": "must be a valid UUID"})
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE report_templates
		SET deleted_at = $2, enabled = false, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL`, templateID, deletedAt)
	if err != nil {
		return fmt.Errorf("delete report template: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return service.NewError(service.CodeNotFound, "report template not found", pgx.ErrNoRows)
	}
	return nil
}

func (r *PostgresRepository) GetReportTemplateStructure(ctx context.Context, id string) (service.ReportTemplateStructure, error) {
	templateID, err := parseUUID(id)
	if err != nil {
		return service.ReportTemplateStructure{}, service.ValidationError(map[string]string{"reportTemplateId": "must be a valid UUID"})
	}
	var structure, style []byte
	if err := r.db.QueryRow(ctx, `
		SELECT structure_json, style_config_json
		FROM report_templates
		WHERE id = $1 AND deleted_at IS NULL`, templateID).Scan(&structure, &style); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.ReportTemplateStructure{}, service.NewError(service.CodeNotFound, "report template not found", err)
		}
		return service.ReportTemplateStructure{}, fmt.Errorf("get report template structure: %w", err)
	}
	return service.ReportTemplateStructure{OutlineSchema: structure, StyleConfig: style}, nil
}

func (r *PostgresRepository) UpdateReportTemplateStructure(ctx context.Context, id string, structure service.ReportTemplateStructure, updatedAt time.Time) (service.ReportTemplateStructure, error) {
	templateID, err := parseUUID(id)
	if err != nil {
		return service.ReportTemplateStructure{}, service.ValidationError(map[string]string{"reportTemplateId": "must be a valid UUID"})
	}
	var outline, style []byte
	if err := r.db.QueryRow(ctx, `
		UPDATE report_templates
		SET structure_json = $2::jsonb, style_config_json = $3::jsonb, updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING structure_json, style_config_json`,
		templateID,
		string(structure.OutlineSchema),
		string(structure.StyleConfig),
		updatedAt,
	).Scan(&outline, &style); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.ReportTemplateStructure{}, service.NewError(service.CodeNotFound, "report template not found", err)
		}
		return service.ReportTemplateStructure{}, fmt.Errorf("update report template structure: %w", err)
	}
	return service.ReportTemplateStructure{OutlineSchema: outline, StyleConfig: style}, nil
}

func (r *PostgresRepository) ListReportMaterials(ctx context.Context, filter service.ReportMaterialListFilter) (service.ReportMaterialListResult, error) {
	where := []string{"deleted_at IS NULL"}
	args := []any{}
	if strings.TrimSpace(filter.Category) != "" {
		args = append(args, strings.TrimSpace(filter.Category))
		where = append(where, fmt.Sprintf("category = $%d", len(args)))
	}
	if filter.Enabled != nil {
		args = append(args, *filter.Enabled)
		where = append(where, fmt.Sprintf("enabled = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	countQuery := "SELECT count(*) FROM report_materials WHERE " + whereSQL
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return service.ReportMaterialListResult{}, fmt.Errorf("count report materials: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	queryArgs := append(append([]any{}, args...), filter.PageSize, offset)
	query := fmt.Sprintf(`
		SELECT
			id::text, material_name, material_type, COALESCE(category, ''), COALESCE(file_ref, ''),
			filename, file_size, COALESCE(description, ''), tags_json, enabled, COALESCE(created_by, ''),
			created_at, updated_at, deleted_at
		FROM report_materials
		WHERE %s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d OFFSET $%d`, whereSQL, len(queryArgs)-1, len(queryArgs))
	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return service.ReportMaterialListResult{}, fmt.Errorf("list report materials: %w", err)
	}
	defer rows.Close()

	items := []service.ReportMaterial{}
	for rows.Next() {
		item, err := scanReportMaterial(rows)
		if err != nil {
			return service.ReportMaterialListResult{}, fmt.Errorf("scan report material: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return service.ReportMaterialListResult{}, fmt.Errorf("iterate report materials: %w", err)
	}
	return service.ReportMaterialListResult{
		Items: items,
		Page:  service.PageMeta{Page: filter.Page, PageSize: filter.PageSize, Total: int(total)},
	}, nil
}

func (r *PostgresRepository) CreateReportMaterial(ctx context.Context, value service.ReportMaterial) (service.ReportMaterial, error) {
	tags, err := json.Marshal(value.Tags)
	if err != nil {
		return service.ReportMaterial{}, fmt.Errorf("encode material tags: %w", err)
	}
	row := r.db.QueryRow(ctx, `
		INSERT INTO report_materials (
			id, material_name, material_type, category, file_ref, filename, file_size,
			description, tags_json, enabled, created_by, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6, $7,
			NULLIF($8, ''), $9::jsonb, $10, NULLIF($11, ''), $12, $13
		)
		RETURNING
			id::text, material_name, material_type, COALESCE(category, ''), COALESCE(file_ref, ''),
			filename, file_size, COALESCE(description, ''), tags_json, enabled, COALESCE(created_by, ''),
			created_at, updated_at, deleted_at`,
		value.ID,
		value.MaterialName,
		value.MaterialType,
		value.Category,
		value.FileRef,
		value.Filename,
		value.FileSize,
		value.Description,
		string(tags),
		value.Enabled,
		value.CreatedBy,
		value.CreatedAt,
		value.UpdatedAt,
	)
	created, err := scanReportMaterial(row)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ReportMaterial{}, service.NewError(service.CodeConflict, "report material already exists", err)
		}
		return service.ReportMaterial{}, fmt.Errorf("insert report material: %w", err)
	}
	return created, nil
}

func (r *PostgresRepository) FindReportMaterialByID(ctx context.Context, id string) (service.ReportMaterial, error) {
	materialID, err := parseUUID(id)
	if err != nil {
		return service.ReportMaterial{}, service.ValidationError(map[string]string{"materialId": "must be a valid UUID"})
	}
	row := r.db.QueryRow(ctx, `
		SELECT
			id::text, material_name, material_type, COALESCE(category, ''), COALESCE(file_ref, ''),
			filename, file_size, COALESCE(description, ''), tags_json, enabled, COALESCE(created_by, ''),
			created_at, updated_at, deleted_at
		FROM report_materials
		WHERE id = $1 AND deleted_at IS NULL`, materialID)
	material, err := scanReportMaterial(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.ReportMaterial{}, service.NewError(service.CodeNotFound, "report material not found", err)
		}
		return service.ReportMaterial{}, fmt.Errorf("find report material: %w", err)
	}
	return material, nil
}

func (r *PostgresRepository) DeleteReportMaterial(ctx context.Context, id string, deletedAt time.Time) error {
	materialID, err := parseUUID(id)
	if err != nil {
		return service.ValidationError(map[string]string{"materialId": "must be a valid UUID"})
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE report_materials
		SET deleted_at = $2, enabled = false, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL`, materialID, deletedAt)
	if err != nil {
		return fmt.Errorf("delete report material: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return service.NewError(service.CodeNotFound, "report material not found", pgx.ErrNoRows)
	}
	return nil
}

func (r *PostgresRepository) CreateReport(ctx context.Context, value service.Report) (service.Report, error) {
	if value.CreatedAt.IsZero() {
		value.CreatedAt = time.Now().UTC()
	}
	if value.UpdatedAt.IsZero() {
		value.UpdatedAt = value.CreatedAt
	}
	templateID, err := parseOptionalUUIDField(value.TemplateID, "templateId")
	if err != nil {
		return service.Report{}, err
	}
	latestJobID, err := parseOptionalUUIDField(value.LatestJobID, "latestJobId")
	if err != nil {
		return service.Report{}, err
	}
	latestReportFileID, err := parseOptionalUUIDField(value.LatestReportFileID, "latestReportFileId")
	if err != nil {
		return service.Report{}, err
	}
	row := r.db.QueryRow(ctx, `
		INSERT INTO reports (
			id, report_name, report_type, template_id, topic, specialty,
			plant_or_business_object, report_year, status, creator_id, creator_name,
			source, latest_job_id, latest_report_file_id, generated_at, exported_at,
			created_at, updated_at
		)
		VALUES (
			$1, $2, $3, NULLIF($4, '')::uuid, $5, NULLIF($6, ''),
			NULLIF($7, ''), NULLIF($8, 0), $9, NULLIF($10, ''), NULLIF($11, ''),
			$12, NULLIF($13, '')::uuid, NULLIF($14, '')::uuid, $15, $16, $17, $18
		)
		RETURNING
			id::text, report_name, report_type, COALESCE(template_id::text, ''), topic,
			COALESCE(specialty, ''), COALESCE(plant_or_business_object, ''),
			COALESCE(report_year, 0), status, COALESCE(creator_id, ''),
			COALESCE(creator_name, ''), source, COALESCE(latest_job_id::text, ''),
			COALESCE(latest_report_file_id::text, ''), generated_at, exported_at,
			created_at, updated_at, deleted_at`,
		value.ID,
		value.Name,
		value.ReportType,
		templateID,
		value.Topic,
		value.Specialty,
		value.BusinessObject,
		value.Year,
		string(value.Status),
		value.CreatorID,
		value.CreatorName,
		value.Source,
		latestJobID,
		latestReportFileID,
		value.GeneratedAt,
		value.ExportedAt,
		value.CreatedAt,
		value.UpdatedAt,
	)
	report, err := scanReport(row)
	if err != nil {
		if isUniqueViolation(err) {
			return service.Report{}, service.NewError(service.CodeConflict, "report already exists", err)
		}
		return service.Report{}, fmt.Errorf("insert report: %w", err)
	}
	return report, nil
}

func (r *PostgresRepository) CreateReportJob(ctx context.Context, value service.ReportJob) (service.ReportJob, error) {
	if value.CreatedAt.IsZero() {
		value.CreatedAt = time.Now().UTC()
	}
	if value.MaxAttempts == 0 {
		value.MaxAttempts = 3
	}
	templateID, err := parseOptionalUUIDField(value.TemplateID, "templateId")
	if err != nil {
		return service.ReportJob{}, err
	}
	row := r.db.QueryRow(ctx, `
		INSERT INTO report_jobs (
			id, request_id, source, job_type, target_type, target_id, asynq_task_id,
			queue_name, report_id, template_id, status, error_code, error_message,
			retry_count, max_attempts, started_at, finished_at, created_at
		)
		VALUES (
			$1, NULLIF($2, ''), $3, $4, $5, $6, NULLIF($7, ''),
			$8, $9, NULLIF($10, '')::uuid, $11, NULLIF($12, ''), NULLIF($13, ''),
			$14, $15, $16, $17, $18
		)
		RETURNING
			id::text, COALESCE(request_id, ''), source, job_type, target_type,
			target_id, COALESCE(asynq_task_id, ''), queue_name, report_id::text,
			COALESCE(template_id::text, ''), status, COALESCE(error_code, ''),
			COALESCE(error_message, ''), retry_count, max_attempts, started_at,
			finished_at, created_at`,
		value.ID,
		value.RequestID,
		value.Source,
		string(value.JobType),
		value.TargetType,
		value.TargetID,
		value.AsynqTaskID,
		value.QueueName,
		value.ReportID,
		templateID,
		string(value.Status),
		value.ErrorCode,
		value.ErrorMessage,
		value.RetryCount,
		value.MaxAttempts,
		value.StartedAt,
		value.FinishedAt,
		value.CreatedAt,
	)
	job, err := scanReportJob(row)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ReportJob{}, service.NewError(service.CodeConflict, "report job already exists", err)
		}
		return service.ReportJob{}, fmt.Errorf("insert report job: %w", err)
	}
	return job, nil
}

func (r *PostgresRepository) FindReportJobByID(ctx context.Context, id string) (service.ReportJob, error) {
	jobID, err := parseUUID(id)
	if err != nil {
		return service.ReportJob{}, service.NewError(service.CodeValidation, "invalid report job id", err)
	}
	row, err := r.queries.GetReportJobByID(ctx, jobID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.ReportJob{}, service.NewError(service.CodeNotFound, "report job not found", err)
		}
		return service.ReportJob{}, fmt.Errorf("find report job: %w", err)
	}
	return reportJobFromSQLC(row), nil
}

func (r *PostgresRepository) UpdateReportJobRetryState(ctx context.Context, id string, retryCount int, status service.JobStatus, updatedAt time.Time) (service.ReportJob, error) {
	jobID, err := parseUUID(id)
	if err != nil {
		return service.ReportJob{}, service.NewError(service.CodeValidation, "invalid report job id", err)
	}
	row := r.db.QueryRow(ctx, `
		UPDATE report_jobs
		SET retry_count = $2,
			status = $3,
			error_code = NULL,
			error_message = NULL,
			started_at = COALESCE(started_at, $4),
			finished_at = $4
		WHERE id = $1
		RETURNING
			id::text, COALESCE(request_id, ''), source, job_type, target_type,
			target_id, COALESCE(asynq_task_id, ''), queue_name, report_id::text,
			COALESCE(template_id::text, ''), status, COALESCE(error_code, ''),
			COALESCE(error_message, ''), retry_count, max_attempts, started_at,
			finished_at, created_at`,
		jobID,
		retryCount,
		string(status),
		updatedAt,
	)
	job, err := scanReportJob(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.ReportJob{}, service.NewError(service.CodeNotFound, "report job not found", err)
		}
		return service.ReportJob{}, fmt.Errorf("update report job retry state: %w", err)
	}
	return job, nil
}

func (r *PostgresRepository) ListReportJobs(ctx context.Context, reportID string) ([]service.ReportJob, error) {
	id, err := parseUUID(reportID)
	if err != nil {
		return nil, service.NewError(service.CodeValidation, "invalid report id", err)
	}
	rows, err := r.db.Query(ctx, `
		SELECT
			id::text, COALESCE(request_id, ''), source, job_type, target_type,
			target_id, COALESCE(asynq_task_id, ''), queue_name, report_id::text,
			COALESCE(template_id::text, ''), status, COALESCE(error_code, ''),
			COALESCE(error_message, ''), retry_count, max_attempts, started_at,
			finished_at, created_at
		FROM report_jobs
		WHERE report_id = $1
		ORDER BY created_at DESC`, id)
	if err != nil {
		return nil, fmt.Errorf("list report jobs: %w", err)
	}
	defer rows.Close()

	jobs := []service.ReportJob{}
	for rows.Next() {
		job, err := scanReportJob(rows)
		if err != nil {
			return nil, fmt.Errorf("scan report job: %w", err)
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate report jobs: %w", err)
	}
	return jobs, nil
}

func (r *PostgresRepository) CreateReportJobAttempt(ctx context.Context, value service.ReportJobAttempt) (service.ReportJobAttempt, error) {
	if value.CreatedAt.IsZero() {
		value.CreatedAt = time.Now().UTC()
	}
	row := r.db.QueryRow(ctx, `
		INSERT INTO report_job_attempts (
			id, job_id, attempt_number, asynq_task_id, trigger_source, reason,
			status, error_code, error_message, started_at, finished_at, created_at
		)
		VALUES (
			$1, $2, $3, NULLIF($4, ''), $5, NULLIF($6, ''),
			$7, NULLIF($8, ''), NULLIF($9, ''), $10, $11, $12
		)
		RETURNING
			id::text, job_id::text, attempt_number, COALESCE(asynq_task_id, ''),
			trigger_source, COALESCE(reason, ''), status, COALESCE(error_code, ''),
			COALESCE(error_message, ''), started_at, finished_at, created_at`,
		value.ID,
		value.JobID,
		value.AttemptNumber,
		value.AsynqTaskID,
		value.TriggerSource,
		value.Reason,
		string(value.Status),
		value.ErrorCode,
		value.ErrorMessage,
		value.StartedAt,
		value.FinishedAt,
		value.CreatedAt,
	)
	attempt, err := scanReportJobAttempt(row)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ReportJobAttempt{}, service.NewError(service.CodeConflict, "report job attempt already exists", err)
		}
		return service.ReportJobAttempt{}, fmt.Errorf("insert report job attempt: %w", err)
	}
	return attempt, nil
}

func (r *PostgresRepository) ListReportJobAttempts(ctx context.Context, jobID string) ([]service.ReportJobAttempt, error) {
	id, err := parseUUID(jobID)
	if err != nil {
		return nil, service.NewError(service.CodeValidation, "invalid report job id", err)
	}
	rows, err := r.db.Query(ctx, `
		SELECT
			id::text, job_id::text, attempt_number, COALESCE(asynq_task_id, ''),
			trigger_source, COALESCE(reason, ''), status, COALESCE(error_code, ''),
			COALESCE(error_message, ''), started_at, finished_at, created_at
		FROM report_job_attempts
		WHERE job_id = $1
		ORDER BY attempt_number DESC`, id)
	if err != nil {
		return nil, fmt.Errorf("list report job attempts: %w", err)
	}
	defer rows.Close()

	attempts := []service.ReportJobAttempt{}
	for rows.Next() {
		attempt, err := scanReportJobAttempt(rows)
		if err != nil {
			return nil, fmt.Errorf("scan report job attempt: %w", err)
		}
		attempts = append(attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate report job attempts: %w", err)
	}
	return attempts, nil
}

func (r *PostgresRepository) CreateReportEvent(ctx context.Context, value service.ReportEvent) (service.ReportEvent, error) {
	if value.CreatedAt.IsZero() {
		value.CreatedAt = time.Now().UTC()
	}
	jobID, err := parseOptionalUUIDField(value.JobID, "jobId")
	if err != nil {
		return service.ReportEvent{}, err
	}
	payloadJSON, err := marshalObject(value.Payload)
	if err != nil {
		return service.ReportEvent{}, err
	}
	row := r.db.QueryRow(ctx, `
		INSERT INTO report_events (
			id, report_id, job_id, event_type, message, payload_json, created_at
		)
		VALUES ($1, $2, NULLIF($3, '')::uuid, $4, NULLIF($5, ''), $6::jsonb, $7)
		RETURNING id::text, report_id::text, COALESCE(job_id::text, ''), event_type, COALESCE(message, ''), payload_json, created_at`,
		value.ID,
		value.ReportID,
		jobID,
		value.EventType,
		value.Message,
		string(payloadJSON),
		value.CreatedAt,
	)
	event, err := scanReportEvent(row)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ReportEvent{}, service.NewError(service.CodeConflict, "report event already exists", err)
		}
		return service.ReportEvent{}, fmt.Errorf("insert report event: %w", err)
	}
	return event, nil
}

func (r *PostgresRepository) ListReportEvents(ctx context.Context, reportID string) ([]service.ReportEvent, error) {
	id, err := parseUUID(reportID)
	if err != nil {
		return nil, service.NewError(service.CodeValidation, "invalid report id", err)
	}
	rows, err := r.db.Query(ctx, `
		SELECT id::text, report_id::text, COALESCE(job_id::text, ''), event_type, COALESCE(message, ''), payload_json, created_at
		FROM report_events
		WHERE report_id = $1
		ORDER BY created_at ASC`, id)
	if err != nil {
		return nil, fmt.Errorf("list report events: %w", err)
	}
	defer rows.Close()

	events := []service.ReportEvent{}
	for rows.Next() {
		event, err := scanReportEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan report event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate report events: %w", err)
	}
	return events, nil
}

func (r *PostgresRepository) CreateReportFile(ctx context.Context, value service.ReportFile) (service.ReportFile, error) {
	reportID, err := parseUUID(value.ReportID)
	if err != nil {
		return service.ReportFile{}, service.NewError(service.CodeValidation, "invalid report id", err)
	}
	jobID, err := parseOptionalUUIDField(value.JobID, "jobId")
	if err != nil {
		return service.ReportFile{}, err
	}
	row := r.db.QueryRow(ctx, `
		INSERT INTO report_files (
			id, report_id, job_id, filename, file_type, file_size, file_status, created_by, created_at
		)
		VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, $6, $7, NULLIF($8, ''), $9)
		RETURNING
			id::text, report_id::text, COALESCE(job_id::text, ''), filename, file_type,
			file_size, file_status, COALESCE(created_by, ''), created_at`,
		value.ID,
		reportID,
		jobID,
		value.Filename,
		value.Format,
		value.FileSize,
		string(value.Status),
		value.CreatedBy,
		value.CreatedAt,
	)
	file, err := scanReportFile(row)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ReportFile{}, service.NewError(service.CodeConflict, "report file already exists", err)
		}
		return service.ReportFile{}, fmt.Errorf("insert report file: %w", err)
	}
	return file, nil
}

func (r *PostgresRepository) SaveReportFileContent(ctx context.Context, reportFileID string, content []byte, createdAt time.Time) error {
	fileID, err := parseUUID(reportFileID)
	if err != nil {
		return service.NewError(service.CodeValidation, "invalid report file id", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO report_file_contents (report_file_id, content, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (report_file_id) DO UPDATE
		SET content = EXCLUDED.content, created_at = EXCLUDED.created_at`,
		fileID,
		content,
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("save report file content: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetReportFileByID(ctx context.Context, id string) (service.ReportFile, error) {
	fileID, err := parseUUID(id)
	if err != nil {
		return service.ReportFile{}, service.NewError(service.CodeValidation, "invalid report file id", err)
	}
	row := r.db.QueryRow(ctx, `
		SELECT
			id::text, report_id::text, COALESCE(job_id::text, ''), filename, file_type,
			file_size, file_status, COALESCE(created_by, ''), created_at
		FROM report_files
		WHERE id = $1`, fileID)
	file, err := scanReportFile(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.ReportFile{}, service.NewError(service.CodeNotFound, "report file not found", err)
		}
		return service.ReportFile{}, fmt.Errorf("get report file: %w", err)
	}
	return file, nil
}

func (r *PostgresRepository) GetReportFileContent(ctx context.Context, reportFileID string) ([]byte, error) {
	fileID, err := parseUUID(reportFileID)
	if err != nil {
		return nil, service.NewError(service.CodeValidation, "invalid report file id", err)
	}
	var content []byte
	err = r.db.QueryRow(ctx, `
		SELECT content
		FROM report_file_contents
		WHERE report_file_id = $1`,
		fileID,
	).Scan(&content)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, service.NewError(service.CodeNotFound, "report file content not found", err)
		}
		return nil, fmt.Errorf("get report file content: %w", err)
	}
	return content, nil
}

func (r *PostgresRepository) ListReportFiles(ctx context.Context, filter service.ReportFileListFilter) ([]service.ReportFile, int, error) {
	conditions := []string{"1 = 1"}
	args := []any{}
	argN := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if strings.TrimSpace(filter.ReportID) != "" {
		reportID, err := parseUUID(filter.ReportID)
		if err != nil {
			return nil, 0, service.NewError(service.CodeValidation, "invalid report id", err)
		}
		conditions = append(conditions, "report_id = "+argN(reportID))
	}
	if strings.TrimSpace(filter.CreatorID) != "" {
		conditions = append(conditions, "report_id IN (SELECT id FROM reports WHERE creator_id = "+argN(filter.CreatorID)+" AND deleted_at IS NULL)")
	}
	where := strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM report_files WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count report files: %w", err)
	}
	limitArg := argN(filter.PageSize)
	offsetArg := argN((filter.Page - 1) * filter.PageSize)
	query := fmt.Sprintf(`
		SELECT
			id::text, report_id::text, COALESCE(job_id::text, ''), filename, file_type,
			file_size, file_status, COALESCE(created_by, ''), created_at
		FROM report_files
		WHERE %s
		ORDER BY created_at DESC
		LIMIT %s OFFSET %s`, where, limitArg, offsetArg)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list report files: %w", err)
	}
	defer rows.Close()

	files := []service.ReportFile{}
	for rows.Next() {
		file, err := scanReportFile(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan report file: %w", err)
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate report files: %w", err)
	}
	return files, total, nil
}

func (r *PostgresRepository) GetReportStatisticsOverview(ctx context.Context, recentDays int) (service.ReportStatisticsOverview, error) {
	var overview service.ReportStatisticsOverview
	overview.RecentDays = recentDays
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM reports WHERE status <> 'deleted'`).Scan(&overview.ReportCount); err != nil {
		return service.ReportStatisticsOverview{}, fmt.Errorf("count reports: %w", err)
	}
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM report_templates WHERE deleted_at IS NULL`).Scan(&overview.TemplateCount); err != nil {
		return service.ReportStatisticsOverview{}, fmt.Errorf("count report templates: %w", err)
	}
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM report_materials WHERE deleted_at IS NULL`).Scan(&overview.MaterialCount); err != nil {
		return service.ReportStatisticsOverview{}, fmt.Errorf("count report materials: %w", err)
	}
	rows, err := r.db.Query(ctx, `SELECT status, count(*) FROM report_jobs GROUP BY status`)
	if err != nil {
		return service.ReportStatisticsOverview{}, fmt.Errorf("count report jobs by status: %w", err)
	}
	defer rows.Close()
	overview.JobStatusCounts = map[string]int{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return service.ReportStatisticsOverview{}, fmt.Errorf("scan job status count: %w", err)
		}
		overview.JobStatusCounts[status] = count
	}
	if err := rows.Err(); err != nil {
		return service.ReportStatisticsOverview{}, fmt.Errorf("iterate job status counts: %w", err)
	}
	return overview, nil
}

func (r *PostgresRepository) ListDailyReportStatistics(ctx context.Context, days int) ([]service.ReportDailyStatistic, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			created_at::date::text AS day,
			report_type,
			count(*) AS created_count,
			count(*) FILTER (WHERE status IN ('generated', 'exported')) AS generated_count,
			count(*) FILTER (WHERE status = 'failed') AS failed_count,
			count(*) FILTER (WHERE status = 'exported') AS exported_count
		FROM reports
		WHERE created_at >= (CURRENT_DATE - ($1::int - 1) * INTERVAL '1 day')
			AND status <> 'deleted'
		GROUP BY day, report_type
		ORDER BY day ASC, report_type ASC`, days)
	if err != nil {
		return nil, fmt.Errorf("list daily report statistics: %w", err)
	}
	defer rows.Close()

	items := []service.ReportDailyStatistic{}
	for rows.Next() {
		var item service.ReportDailyStatistic
		if err := rows.Scan(&item.Date, &item.ReportType, &item.CreatedCount, &item.GeneratedCount, &item.FailedCount, &item.ExportedCount); err != nil {
			return nil, fmt.Errorf("scan daily report statistic: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate daily report statistics: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) ListReportOperationLogs(ctx context.Context, filter service.ReportOperationLogFilter) ([]service.ReportOperationLog, int, error) {
	conditions := []string{"1 = 1"}
	args := []any{}
	argN := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.TargetType != "" {
		conditions = append(conditions, "target_type = "+argN(filter.TargetType))
	}
	if filter.TargetID != "" {
		conditions = append(conditions, "target_id = "+argN(filter.TargetID))
	}
	if filter.OperationType != "" {
		conditions = append(conditions, "operation_type = "+argN(filter.OperationType))
	}
	if filter.RequestID != "" {
		conditions = append(conditions, "request_id = "+argN(filter.RequestID))
	}
	if filter.RequestSource != "" {
		conditions = append(conditions, "request_source = "+argN(filter.RequestSource))
	}
	if filter.ToolName != "" {
		conditions = append(conditions, "tool_name = "+argN(filter.ToolName))
	}
	where := strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM report_operation_logs WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count report operation logs: %w", err)
	}
	limitArg := argN(filter.PageSize)
	offsetArg := argN((filter.Page - 1) * filter.PageSize)
	query := fmt.Sprintf(`
		SELECT
			id::text, COALESCE(operator_id, ''), COALESCE(operator_name, ''),
			operation_type, target_type, target_id, COALESCE(request_id, ''),
			COALESCE(request_source, ''), COALESCE(tool_name, ''), parameter_summary_json,
			operation_result, COALESCE(error_message, ''), metadata_json, created_at
		FROM report_operation_logs
		WHERE %s
		ORDER BY created_at DESC
		LIMIT %s OFFSET %s`, where, limitArg, offsetArg)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list report operation logs: %w", err)
	}
	defer rows.Close()

	logs := []service.ReportOperationLog{}
	for rows.Next() {
		item, err := scanReportOperationLog(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan report operation log: %w", err)
		}
		logs = append(logs, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate report operation logs: %w", err)
	}
	return logs, total, nil
}

func (r *PostgresRepository) GetReportSettings(ctx context.Context) (service.ReportSettings, error) {
	var raw []byte
	err := r.db.QueryRow(ctx, `SELECT settings_json FROM report_settings WHERE id = 'default'`).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultReportSettings(), nil
		}
		return service.ReportSettings{}, fmt.Errorf("get report settings: %w", err)
	}
	settings, err := decodeReportSettings(raw)
	if err != nil {
		return service.ReportSettings{}, err
	}
	return settings, nil
}

func (r *PostgresRepository) UpdateReportSettings(ctx context.Context, value service.ReportSettings, updatedBy string, updatedAt time.Time) (service.UpdateReportSettingsResult, error) {
	raw, err := encodeReportSettings(value)
	if err != nil {
		return service.UpdateReportSettingsResult{}, err
	}
	var result service.UpdateReportSettingsResult
	if err := r.db.QueryRow(ctx, `
		INSERT INTO report_settings (id, settings_json, updated_by, updated_at)
		VALUES ('default', $1::jsonb, NULLIF($2, ''), $3)
		ON CONFLICT (id) DO UPDATE SET
			settings_json = EXCLUDED.settings_json,
			updated_by = EXCLUDED.updated_by,
			updated_at = EXCLUDED.updated_at
		RETURNING updated_at`, string(raw), updatedBy, updatedAt).Scan(&result.UpdatedAt); err != nil {
		return service.UpdateReportSettingsResult{}, fmt.Errorf("update report settings: %w", err)
	}
	return result, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanReportType(row scanner) (service.ReportType, error) {
	var value service.ReportType
	if err := row.Scan(
		&value.Code,
		&value.Name,
		&value.Description,
		&value.Enabled,
		&value.DefaultTemplateID,
		&value.CreatedAt,
		&value.UpdatedAt,
	); err != nil {
		return service.ReportType{}, err
	}
	return value, nil
}

func scanReport(row scanner) (service.Report, error) {
	var value service.Report
	var status string
	if err := row.Scan(
		&value.ID,
		&value.Name,
		&value.ReportType,
		&value.TemplateID,
		&value.Topic,
		&value.Specialty,
		&value.BusinessObject,
		&value.Year,
		&status,
		&value.CreatorID,
		&value.CreatorName,
		&value.Source,
		&value.LatestJobID,
		&value.LatestReportFileID,
		&value.GeneratedAt,
		&value.ExportedAt,
		&value.CreatedAt,
		&value.UpdatedAt,
		&value.DeletedAt,
	); err != nil {
		return service.Report{}, err
	}
	value.Status = service.ReportStatus(status)
	return value, nil
}

func scanReportTemplate(row scanner) (service.ReportTemplate, error) {
	var value service.ReportTemplate
	if err := row.Scan(
		&value.ID,
		&value.TemplateName,
		&value.ReportType,
		&value.Version,
		&value.FileRef,
		&value.Filename,
		&value.FileSize,
		&value.Description,
		&value.Enabled,
		&value.CreatedBy,
		&value.CreatedAt,
		&value.UpdatedAt,
		&value.DeletedAt,
	); err != nil {
		return service.ReportTemplate{}, err
	}
	return value, nil
}

func scanReportMaterial(row scanner) (service.ReportMaterial, error) {
	var value service.ReportMaterial
	var tagsRaw []byte
	if err := row.Scan(
		&value.ID,
		&value.MaterialName,
		&value.MaterialType,
		&value.Category,
		&value.FileRef,
		&value.Filename,
		&value.FileSize,
		&value.Description,
		&tagsRaw,
		&value.Enabled,
		&value.CreatedBy,
		&value.CreatedAt,
		&value.UpdatedAt,
		&value.DeletedAt,
	); err != nil {
		return service.ReportMaterial{}, err
	}
	if len(tagsRaw) > 0 {
		if err := json.Unmarshal(tagsRaw, &value.Tags); err != nil {
			return service.ReportMaterial{}, err
		}
	}
	if value.Tags == nil {
		value.Tags = []string{}
	}
	return value, nil
}

func scanReportJob(row scanner) (service.ReportJob, error) {
	var value service.ReportJob
	var jobType, status string
	if err := row.Scan(
		&value.ID,
		&value.RequestID,
		&value.Source,
		&jobType,
		&value.TargetType,
		&value.TargetID,
		&value.AsynqTaskID,
		&value.QueueName,
		&value.ReportID,
		&value.TemplateID,
		&status,
		&value.ErrorCode,
		&value.ErrorMessage,
		&value.RetryCount,
		&value.MaxAttempts,
		&value.StartedAt,
		&value.FinishedAt,
		&value.CreatedAt,
	); err != nil {
		return service.ReportJob{}, err
	}
	value.JobType = service.JobType(jobType)
	value.Status = service.JobStatus(status)
	return value, nil
}

func scanReportJobAttempt(row scanner) (service.ReportJobAttempt, error) {
	var value service.ReportJobAttempt
	var status string
	if err := row.Scan(
		&value.ID,
		&value.JobID,
		&value.AttemptNumber,
		&value.AsynqTaskID,
		&value.TriggerSource,
		&value.Reason,
		&status,
		&value.ErrorCode,
		&value.ErrorMessage,
		&value.StartedAt,
		&value.FinishedAt,
		&value.CreatedAt,
	); err != nil {
		return service.ReportJobAttempt{}, err
	}
	value.Status = service.JobStatus(status)
	return value, nil
}

func scanReportEvent(row scanner) (service.ReportEvent, error) {
	var value service.ReportEvent
	var payloadJSON []byte
	if err := row.Scan(
		&value.ID,
		&value.ReportID,
		&value.JobID,
		&value.EventType,
		&value.Message,
		&payloadJSON,
		&value.CreatedAt,
	); err != nil {
		return service.ReportEvent{}, err
	}
	if len(payloadJSON) > 0 {
		if err := json.Unmarshal(payloadJSON, &value.Payload); err != nil {
			return service.ReportEvent{}, fmt.Errorf("unmarshal report event payload: %w", err)
		}
	}
	return value, nil
}

func scanReportFile(row scanner) (service.ReportFile, error) {
	var value service.ReportFile
	var status string
	if err := row.Scan(
		&value.ID,
		&value.ReportID,
		&value.JobID,
		&value.Filename,
		&value.Format,
		&value.FileSize,
		&status,
		&value.CreatedBy,
		&value.CreatedAt,
	); err != nil {
		return service.ReportFile{}, err
	}
	value.Status = service.JobStatus(status)
	value.ContentPath = "/api/v1/report-files/" + value.ID + "/content"
	return value, nil
}

func scanReportOperationLog(row scanner) (service.ReportOperationLog, error) {
	var value service.ReportOperationLog
	var parameterSummaryJSON, metadataJSON []byte
	if err := row.Scan(
		&value.ID,
		&value.OperatorID,
		&value.OperatorName,
		&value.OperationType,
		&value.TargetType,
		&value.TargetID,
		&value.RequestID,
		&value.RequestSource,
		&value.ToolName,
		&parameterSummaryJSON,
		&value.OperationResult,
		&value.ErrorMessage,
		&metadataJSON,
		&value.CreatedAt,
	); err != nil {
		return service.ReportOperationLog{}, err
	}
	if len(parameterSummaryJSON) > 0 {
		if err := json.Unmarshal(parameterSummaryJSON, &value.ParameterSummary); err != nil {
			return service.ReportOperationLog{}, fmt.Errorf("unmarshal operation parameter summary: %w", err)
		}
	}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &value.Metadata); err != nil {
			return service.ReportOperationLog{}, fmt.Errorf("unmarshal operation metadata: %w", err)
		}
	}
	return value, nil
}

func reportJobFromSQLC(row sqlc.GetReportJobByIDRow) service.ReportJob {
	return service.ReportJob{
		ID:           uuidToString(row.ID),
		RequestID:    textToString(row.RequestID),
		Source:       row.Source,
		JobType:      service.JobType(row.JobType),
		TargetType:   row.TargetType,
		TargetID:     row.TargetID,
		AsynqTaskID:  textToString(row.AsynqTaskID),
		QueueName:    row.QueueName,
		ReportID:     uuidToString(row.ReportID),
		TemplateID:   uuidToString(row.TemplateID),
		Status:       service.JobStatus(row.Status),
		ErrorCode:    textToString(row.ErrorCode),
		ErrorMessage: textToString(row.ErrorMessage),
		RetryCount:   int(row.RetryCount),
		MaxAttempts:  int(row.MaxAttempts),
		StartedAt:    timestamptzToTimePtr(row.StartedAt),
		FinishedAt:   timestamptzToTimePtr(row.FinishedAt),
		CreatedAt:    timestamptzToTime(row.CreatedAt),
	}
}

func parseUUID(value string) (pgtype.UUID, error) {
	var uuid pgtype.UUID
	if err := uuid.Scan(value); err != nil {
		return pgtype.UUID{}, err
	}
	return uuid, nil
}

func parseOptionalUUIDField(value, field string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if _, err := parseUUID(trimmed); err != nil {
		return "", service.ValidationError(map[string]string{field: "must be a valid UUID"})
	}
	return trimmed, nil
}

func uuidToString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	b := value.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func textToString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func timestamptzToTime(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func timestamptzToTimePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func marshalObject(value map[string]any) ([]byte, error) {
	if value == nil {
		return []byte("{}"), nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal json object: %w", err)
	}
	return data, nil
}

func defaultReportSettings() service.ReportSettings {
	return service.ReportSettings{
		LLM: map[string]any{
			"provider": "ai-gateway",
		},
		DefaultTemplates: map[string]string{},
		File: map[string]any{
			"defaultFormat":        "docx",
			"defaultNumberingMode": "global",
		},
	}
}

func encodeReportSettings(value service.ReportSettings) ([]byte, error) {
	payload := map[string]any{
		"llm":              value.LLM,
		"defaultTemplates": value.DefaultTemplates,
		"file":             value.File,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal report settings: %w", err)
	}
	return data, nil
}

func decodeReportSettings(data []byte) (service.ReportSettings, error) {
	var payload struct {
		LLM              map[string]any    `json:"llm"`
		DefaultTemplates map[string]string `json:"defaultTemplates"`
		File             map[string]any    `json:"file"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return service.ReportSettings{}, fmt.Errorf("unmarshal report settings: %w", err)
	}
	settings := defaultReportSettings()
	if payload.LLM != nil {
		settings.LLM = payload.LLM
	}
	if payload.DefaultTemplates != nil {
		settings.DefaultTemplates = payload.DefaultTemplates
	}
	if payload.File != nil {
		settings.File = payload.File
	}
	return settings, nil
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "duplicate key value") || strings.Contains(err.Error(), "SQLSTATE 23505")
}
