package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"resumebank/internal/models"
)

type FileRepo struct {
	pool *pgxpool.Pool
}

func NewFileRepo(pool *pgxpool.Pool) *FileRepo {
	return &FileRepo{pool: pool}
}

const fileColumns = `id, candidate_id, kind, original_filename, stored_path, content_type, size_bytes, position, created_at`

func scanFile(row pgx.Row) (*models.CandidateFile, error) {
	f := &models.CandidateFile{}
	err := row.Scan(&f.ID, &f.CandidateID, &f.Kind, &f.OriginalFilename, &f.StoredPath,
		&f.ContentType, &f.SizeBytes, &f.Position, &f.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (r *FileRepo) Create(ctx context.Context, f *models.CandidateFile) (*models.CandidateFile, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO candidate_files (candidate_id, kind, original_filename, stored_path, content_type, size_bytes, position)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING `+fileColumns,
		f.CandidateID, f.Kind, f.OriginalFilename, f.StoredPath, f.ContentType, f.SizeBytes, f.Position,
	)
	return scanFile(row)
}

func (r *FileRepo) GetByID(ctx context.Context, id int64) (*models.CandidateFile, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+fileColumns+` FROM candidate_files WHERE id = $1`, id)
	return scanFile(row)
}

func (r *FileRepo) ListByCandidate(ctx context.Context, candidateID int64) ([]models.CandidateFile, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+fileColumns+` FROM candidate_files WHERE candidate_id = $1 ORDER BY kind, position`, candidateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var files []models.CandidateFile
	for rows.Next() {
		f := models.CandidateFile{}
		if err := rows.Scan(&f.ID, &f.CandidateID, &f.Kind, &f.OriginalFilename, &f.StoredPath,
			&f.ContentType, &f.SizeBytes, &f.Position, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (r *FileRepo) CountByKind(ctx context.Context, candidateID int64, kind models.FileKind) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM candidate_files WHERE candidate_id = $1 AND kind = $2`, candidateID, kind,
	).Scan(&count)
	return count, err
}

func (r *FileRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM candidate_files WHERE id = $1`, id)
	return err
}
