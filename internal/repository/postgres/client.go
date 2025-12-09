package postgres

import (
	"TrackMe/internal/domain/client"
	"TrackMe/internal/domain/contract"
	"TrackMe/pkg/store"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ClientEntity wraps client.Entity and holds raw JSON contracts
type ClientEntity struct {
	client.Entity
	ContractsRaw []byte
}

type ClientRepository struct {
	db *pgxpool.Pool
}

func NewClientRepository(db *pgxpool.Pool) *ClientRepository {
	return &ClientRepository{db: db}
}

// List retrieves all clients from the database.
func (r *ClientRepository) List(ctx context.Context, filters client.Filters, limit, offset int) ([]client.Entity, int, error) {
	query := `SELECT id, name, email, current_stage, last_updated, is_active, source, channel, app, last_login, contracts FROM clients WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM clients WHERE 1=1`

	args := []interface{}{}
	argCount := 1

	// Apply filters
	if filters.ID != "" {
		query += fmt.Sprintf(" AND id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND id = $%d", argCount)
		args = append(args, filters.ID)
		argCount++
	}

	if filters.Stage != "" {
		query += fmt.Sprintf(" AND current_stage = $%d", argCount)
		countQuery += fmt.Sprintf(" AND current_stage = $%d", argCount)
		args = append(args, filters.Stage)
		argCount++
	}

	if filters.Source != "" {
		query += fmt.Sprintf(" AND source = $%d", argCount)
		countQuery += fmt.Sprintf(" AND source = $%d", argCount)
		args = append(args, filters.Source)
		argCount++
	}

	if filters.Channel != "" {
		query += fmt.Sprintf(" AND channel = $%d", argCount)
		countQuery += fmt.Sprintf(" AND channel = $%d", argCount)
		args = append(args, filters.Channel)
		argCount++
	}

	if filters.AppStatus != "" {
		query += fmt.Sprintf(" AND app = $%d", argCount)
		countQuery += fmt.Sprintf(" AND app = $%d", argCount)
		args = append(args, filters.AppStatus)
		argCount++
	}

	if filters.IsActive != nil {
		query += fmt.Sprintf(" AND is_active = $%d", argCount)
		countQuery += fmt.Sprintf(" AND is_active = $%d", argCount)
		args = append(args, *filters.IsActive)
		argCount++
	}

	if !filters.UpdatedAfter.IsZero() {
		query += fmt.Sprintf(" AND last_updated >= $%d", argCount)
		countQuery += fmt.Sprintf(" AND last_updated >= $%d", argCount)
		args = append(args, filters.UpdatedAfter)
		argCount++
	}

	if !filters.LastLoginAfter.IsZero() {
		query += fmt.Sprintf(" AND last_login >= $%d", argCount)
		countQuery += fmt.Sprintf(" AND last_login >= $%d", argCount)
		args = append(args, filters.LastLoginAfter)
		argCount++
	}

	// Get total count
	var total int
	row := r.db.QueryRow(ctx, countQuery, args...)
	if err := row.Scan(&total); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 10
	}

	query += fmt.Sprintf(" ORDER BY last_updated DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var clients []client.Entity
	for rows.Next() {
		var temp ClientEntity
		err := rows.Scan(
			&temp.ID,
			&temp.Name,
			&temp.Email,
			&temp.CurrentStage,
			&temp.LastUpdated,
			&temp.IsActive,
			&temp.Source,
			&temp.Channel,
			&temp.App,
			&temp.LastLogin,
			&temp.ContractsRaw,
		)
		if err != nil {
			return nil, 0, err
		}

		if temp.ContractsRaw != nil {
			var contracts []contract.Entity
			if err := json.Unmarshal(temp.ContractsRaw, &contracts); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal contracts: %w", err)
			}
			temp.Entity.Contracts = contracts
		}

		clients = append(clients, temp.Entity)
	}

	return clients, total, nil
}

// Create inserts a new client into the database.
func (r *ClientRepository) Create(ctx context.Context, data client.Entity) (client.Entity, error) {
	if data.ID == "" {
		data.ID = uuid.NewString()
	}

	if data.CurrentStage == nil {
		stage := "new"
		data.CurrentStage = &stage
	}
	if data.IsActive == nil {
		active := true
		data.IsActive = &active
	}

	var contractsJSON []byte
	if data.Contracts != nil {
		var err error
		contractsJSON, err = json.Marshal(data.Contracts)
		if err != nil {
			return client.Entity{}, fmt.Errorf("failed to marshal contracts: %w", err)
		}
	}

	query := `INSERT INTO clients (
		id, name, email, current_stage, is_active, 
		source, channel, app, last_login, contracts
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	  RETURNING id, last_updated`

	args := []interface{}{
		data.ID,
		data.Name,
		data.Email,
		data.CurrentStage,
		data.IsActive,
		data.Source,
		data.Channel,
		data.App,
		data.LastLogin,
		contractsJSON,
	}

	var temp ClientEntity
	err := r.db.QueryRow(ctx, query, args...).Scan(&temp.ID, &temp.LastUpdated)
	if err != nil {
		return client.Entity{}, fmt.Errorf("failed to insert client: %w", err)
	}

	data.ID = temp.ID
	data.LastUpdated = temp.LastUpdated
	return data, nil
}

// Count counts clients based on a BSON filter.
func (r *ClientRepository) Count(ctx context.Context, filter bson.M) (int64, error) {
	query := "SELECT COUNT(*) FROM clients WHERE 1=1"
	args := []interface{}{}
	argCount := 1

	for key, value := range filter {
		// handle simple equality or Mongo-style operators if needed
		if subMap, ok := value.(bson.M); ok {
			for op, opValue := range subMap {
				switch op {
				case "$gte":
					query += fmt.Sprintf(" AND %s >= $%d", key, argCount)
					args = append(args, opValue)
					argCount++
				case "$lte":
					query += fmt.Sprintf(" AND %s <= $%d", key, argCount)
					args = append(args, opValue)
					argCount++
				case "$ne":
					query += fmt.Sprintf(" AND %s != $%d", key, argCount)
					args = append(args, opValue)
					argCount++
					// add other operators if needed
				}
			}
		} else {
			query += fmt.Sprintf(" AND %s = $%d", key, argCount)
			args = append(args, value)
			argCount++
		}
	}

	var count int64
	row := r.db.QueryRow(ctx, query, args...)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// Get retrieves a client by ID.
func (r *ClientRepository) Get(ctx context.Context, id string) (client.Entity, error) {
	query := `SELECT id, name, email, current_stage, last_updated,
		is_active, source, channel, app, last_login, contracts 
		FROM clients WHERE id=$1`

	var temp ClientEntity
	err := r.db.QueryRow(ctx, query, id).Scan(
		&temp.ID,
		&temp.Name,
		&temp.Email,
		&temp.CurrentStage,
		&temp.LastUpdated,
		&temp.IsActive,
		&temp.Source,
		&temp.Channel,
		&temp.App,
		&temp.LastLogin,
		&temp.ContractsRaw,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return client.Entity{}, store.ErrorNotFound
		}
		return client.Entity{}, err
	}

	if temp.ContractsRaw != nil {
		var contracts []contract.Entity
		if err := json.Unmarshal(temp.ContractsRaw, &contracts); err != nil {
			return client.Entity{}, fmt.Errorf("failed to unmarshal contracts: %w", err)
		}
		temp.Entity.Contracts = contracts
	}

	return temp.Entity, nil
}

// GetByEmail retrieves a client by email.
func (r *ClientRepository) GetByEmail(ctx context.Context, email string) (client.Entity, error) {
	query := `SELECT id, name, email, current_stage, last_updated,
		is_active, source, channel, app, last_login, contracts 
		FROM clients WHERE email=$1`

	var temp ClientEntity
	err := r.db.QueryRow(ctx, query, email).Scan(
		&temp.ID,
		&temp.Name,
		&temp.Email,
		&temp.CurrentStage,
		&temp.LastUpdated,
		&temp.IsActive,
		&temp.Source,
		&temp.Channel,
		&temp.App,
		&temp.LastLogin,
		&temp.ContractsRaw,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return client.Entity{}, store.ErrorNotFound
		}
		return client.Entity{}, err
	}

	if temp.ContractsRaw != nil {
		var contracts []contract.Entity
		if err := json.Unmarshal(temp.ContractsRaw, &contracts); err != nil {
			return client.Entity{}, fmt.Errorf("failed to unmarshal contracts: %w", err)
		}
		temp.Entity.Contracts = contracts
	}

	return temp.Entity, nil
}

// Update modifies an existing client.
func (r *ClientRepository) Update(ctx context.Context, id string, data client.Entity) (client.Entity, error) {
	query := `UPDATE clients SET 
		name=$1, email=$2, current_stage=$3, is_active=$4,
		source=$5, channel=$6, app=$7, last_login=$8, contracts=$9,
		last_updated=NOW()
		WHERE id=$10
		RETURNING id, name, email, current_stage, last_updated,
		is_active, source, channel, app, last_login, contracts`

	var contractsJSON []byte
	if data.Contracts != nil {
		var err error
		contractsJSON, err = json.Marshal(data.Contracts)
		if err != nil {
			return client.Entity{}, fmt.Errorf("failed to marshal contracts: %w", err)
		}
	}

	args := []interface{}{
		data.Name,
		data.Email,
		data.CurrentStage,
		data.IsActive,
		data.Source,
		data.Channel,
		data.App,
		data.LastLogin,
		contractsJSON,
		id,
	}

	var temp ClientEntity
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&temp.ID,
		&temp.Name,
		&temp.Email,
		&temp.CurrentStage,
		&temp.LastUpdated,
		&temp.IsActive,
		&temp.Source,
		&temp.Channel,
		&temp.App,
		&temp.LastLogin,
		&temp.ContractsRaw,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return client.Entity{}, store.ErrorNotFound
		}
		return client.Entity{}, err
	}

	if temp.ContractsRaw != nil {
		var contracts []contract.Entity
		if err := json.Unmarshal(temp.ContractsRaw, &contracts); err != nil {
			return client.Entity{}, fmt.Errorf("failed to unmarshal contracts: %w", err)
		}
		temp.Entity.Contracts = contracts
	}

	return temp.Entity, nil
}

// Delete removes a client.
func (r *ClientRepository) Delete(ctx context.Context, id string) error {
	cmdTag, err := r.db.Exec(ctx, "DELETE FROM clients WHERE id=$1", id)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return store.ErrorNotFound
	}
	return nil
}
