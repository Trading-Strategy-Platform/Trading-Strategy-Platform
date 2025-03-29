// services/strategy-service/internal/repository/strategy_repository.go
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"services/strategy-service/internal/model"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

// StrategyRepository handles database operations for strategies
type StrategyRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewStrategyRepository creates a new strategy repository
func NewStrategyRepository(db *sqlx.DB, logger *zap.Logger) *StrategyRepository {
	return &StrategyRepository{
		db:     db,
		logger: logger,
	}
}

func (r *StrategyRepository) GetUserStrategies(ctx context.Context, userID int, searchTerm string, purchasedOnly bool, tags []int) ([]model.Strategy, error) {
	query := `SELECT * FROM get_my_strategies($1, $2, $3, $4)`

	var strategies []model.Strategy
	err := r.db.SelectContext(ctx, &strategies, query, userID, searchTerm, purchasedOnly, pq.Array(tags))
	if err != nil {
		r.logger.Error("Failed to get user strategies", zap.Error(err))
		return nil, err
	}

	return strategies, nil
}

// Create adds a new strategy to the database
func (r *StrategyRepository) Create(ctx context.Context, tx *sqlx.Tx, strategy *model.StrategyCreate, userID int) (int, error) {
	query := `
		INSERT INTO strategies (name, user_id, description, structure, is_public, version, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	// Convert strategy structure to JSON
	structureBytes, err := json.Marshal(strategy.Structure)
	if err != nil {
		r.logger.Error("Failed to marshal strategy structure", zap.Error(err))
		return 0, err
	}

	var id int
	var execContext interface {
		QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	}

	if tx != nil {
		execContext = tx
	} else {
		execContext = r.db
	}

	err = execContext.QueryRowContext(
		ctx,
		query,
		strategy.Name,
		userID,
		strategy.Description,
		structureBytes,
		strategy.IsPublic,
		1, // Initial version
		time.Now(),
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to create strategy", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// GetByID retrieves a strategy by ID
func (r *StrategyRepository) GetByID(ctx context.Context, id int) (*model.Strategy, error) {
	query := `
		SELECT id, name, user_id, description, structure, is_public, version, created_at, updated_at
		FROM strategies
		WHERE id = $1
	`

	var strategy model.Strategy
	var structureBytes []byte

	row := r.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&strategy.ID,
		&strategy.Name,
		&strategy.UserID,
		&strategy.Description,
		&structureBytes,
		&strategy.IsPublic,
		&strategy.Version,
		&strategy.CreatedAt,
		&strategy.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get strategy by ID", zap.Error(err), zap.Int("strategy_id", id))
		return nil, err
	}

	// Unmarshal the strategy structure
	if err := json.Unmarshal(structureBytes, &strategy.Structure); err != nil {
		r.logger.Error("Failed to unmarshal strategy structure", zap.Error(err))
		return nil, err
	}

	// Get tags for the strategy
	tags, err := r.getStrategyTags(ctx, id)
	if err != nil {
		r.logger.Warn("Failed to get strategy tags", zap.Error(err), zap.Int("strategy_id", id))
	} else {
		strategy.Tags = tags
	}

	return &strategy, nil
}

// GetByUserID retrieves all strategies for a user
func (r *StrategyRepository) GetByUserID(ctx context.Context, userID int, isPublic *bool, tagID *int, page, limit int) ([]model.Strategy, int, error) {
	query := `
		SELECT id, name, user_id, description, structure, is_public, version, created_at, updated_at
		FROM strategies
		WHERE user_id = $1
	`

	countQuery := `
		SELECT COUNT(*)
		FROM strategies
		WHERE user_id = $1
	`

	params := []interface{}{userID}
	paramIndex := 2

	// Add isPublic filter if provided
	if isPublic != nil {
		query += fmt.Sprintf(" AND is_public = $%d", paramIndex)
		countQuery += fmt.Sprintf(" AND is_public = $%d", paramIndex)
		params = append(params, *isPublic)
		paramIndex++
	}

	// Add tag filter if provided
	if tagID != nil {
		query += fmt.Sprintf(" AND id IN (SELECT strategy_id FROM strategy_tag_mappings WHERE tag_id = $%d)", paramIndex)
		countQuery += fmt.Sprintf(" AND id IN (SELECT strategy_id FROM strategy_tag_mappings WHERE tag_id = $%d)", paramIndex)
		params = append(params, *tagID)
		paramIndex++
	}

	// Add pagination
	offset := (page - 1) * limit
	query += " ORDER BY created_at DESC LIMIT $" + fmt.Sprintf("%d", paramIndex) + " OFFSET $" + fmt.Sprintf("%d", paramIndex+1)
	params = append(params, limit, offset)

	// Execute count query first
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, params[:paramIndex-1]...)
	if err != nil {
		r.logger.Error("Failed to count strategies", zap.Error(err))
		return nil, 0, err
	}

	// If count is 0, return empty result
	if total == 0 {
		return []model.Strategy{}, 0, nil
	}

	// Execute main query
	rows, err := r.db.QueryContext(ctx, query, params...)
	if err != nil {
		r.logger.Error("Failed to query strategies", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	strategies := []model.Strategy{}
	for rows.Next() {
		var strategy model.Strategy
		var structureBytes []byte

		err := rows.Scan(
			&strategy.ID,
			&strategy.Name,
			&strategy.UserID,
			&strategy.Description,
			&structureBytes,
			&strategy.IsPublic,
			&strategy.Version,
			&strategy.CreatedAt,
			&strategy.UpdatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan strategy row", zap.Error(err))
			return nil, 0, err
		}

		// Unmarshal the strategy structure
		if err := json.Unmarshal(structureBytes, &strategy.Structure); err != nil {
			r.logger.Error("Failed to unmarshal strategy structure", zap.Error(err))
			return nil, 0, err
		}

		strategies = append(strategies, strategy)
	}

	// Get tags for each strategy
	for i, strategy := range strategies {
		tags, err := r.getStrategyTags(ctx, strategy.ID)
		if err != nil {
			r.logger.Warn("Failed to get strategy tags", zap.Error(err), zap.Int("strategy_id", strategy.ID))
		} else {
			strategies[i].Tags = tags
		}
	}

	return strategies, total, nil
}

// GetPublicStrategies retrieves public strategies with optional filtering
func (r *StrategyRepository) GetPublicStrategies(ctx context.Context, tagID *int, page, limit int) ([]model.Strategy, int, error) {
	query := `
		SELECT id, name, user_id, description, structure, is_public, version, created_at, updated_at
		FROM strategies
		WHERE is_public = true
	`

	countQuery := `
		SELECT COUNT(*)
		FROM strategies
		WHERE is_public = true
	`

	params := []interface{}{}
	paramIndex := 1

	// Add tag filter if provided
	if tagID != nil {
		query += fmt.Sprintf(" AND id IN (SELECT strategy_id FROM strategy_tag_mappings WHERE tag_id = $%d)", paramIndex)
		countQuery += fmt.Sprintf(" AND id IN (SELECT strategy_id FROM strategy_tag_mappings WHERE tag_id = $%d)", paramIndex)
		params = append(params, *tagID)
		paramIndex++
	}

	// Add pagination
	offset := (page - 1) * limit
	query += " ORDER BY created_at DESC LIMIT $" + fmt.Sprintf("%d", paramIndex) + " OFFSET $" + fmt.Sprintf("%d", paramIndex+1)
	params = append(params, limit, offset)

	// Execute count query first
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, params[:paramIndex-1]...)
	if err != nil {
		r.logger.Error("Failed to count public strategies", zap.Error(err))
		return nil, 0, err
	}

	// If count is 0, return empty result
	if total == 0 {
		return []model.Strategy{}, 0, nil
	}

	// Execute main query
	rows, err := r.db.QueryContext(ctx, query, params...)
	if err != nil {
		r.logger.Error("Failed to query public strategies", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	strategies := []model.Strategy{}
	for rows.Next() {
		var strategy model.Strategy
		var structureBytes []byte

		err := rows.Scan(
			&strategy.ID,
			&strategy.Name,
			&strategy.UserID,
			&strategy.Description,
			&structureBytes,
			&strategy.IsPublic,
			&strategy.Version,
			&strategy.CreatedAt,
			&strategy.UpdatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan strategy row", zap.Error(err))
			return nil, 0, err
		}

		// Unmarshal the strategy structure
		if err := json.Unmarshal(structureBytes, &strategy.Structure); err != nil {
			r.logger.Error("Failed to unmarshal strategy structure", zap.Error(err))
			return nil, 0, err
		}

		strategies = append(strategies, strategy)
	}

	// Get tags for each strategy
	for i, strategy := range strategies {
		tags, err := r.getStrategyTags(ctx, strategy.ID)
		if err != nil {
			r.logger.Warn("Failed to get strategy tags", zap.Error(err), zap.Int("strategy_id", strategy.ID))
		} else {
			strategies[i].Tags = tags
		}
	}

	return strategies, total, nil
}

// Update updates a strategy's details
func (r *StrategyRepository) Update(ctx context.Context, tx *sqlx.Tx, id int, update *model.StrategyUpdate) error {
	query := `
		UPDATE strategies
		SET
	`

	params := []interface{}{}
	paramCount := 1
	setValues := []string{}

	if update.Name != nil {
		setValues = append(setValues, fmt.Sprintf("name = $%d", paramCount))
		params = append(params, *update.Name)
		paramCount++
	}

	if update.Description != nil {
		setValues = append(setValues, fmt.Sprintf("description = $%d", paramCount))
		params = append(params, *update.Description)
		paramCount++
	}

	if update.Structure != nil {
		structureBytes, err := json.Marshal(*update.Structure)
		if err != nil {
			r.logger.Error("Failed to marshal strategy structure", zap.Error(err))
			return err
		}

		setValues = append(setValues, fmt.Sprintf("structure = $%d", paramCount))
		params = append(params, structureBytes)
		paramCount++

		// Increment version when structure changes
		setValues = append(setValues, fmt.Sprintf("version = version + 1"))
	}

	if update.IsPublic != nil {
		setValues = append(setValues, fmt.Sprintf("is_public = $%d", paramCount))
		params = append(params, *update.IsPublic)
		paramCount++
	}

	// Always update the updated_at timestamp
	setValues = append(setValues, fmt.Sprintf("updated_at = $%d", paramCount))
	params = append(params, time.Now())
	paramCount++

	// If no fields were provided for update, return
	if len(setValues) == 1 { // Only updated_at
		return nil
	}

	// Combine SET clauses
	for i, setValue := range setValues {
		if i == 0 {
			query += setValue
		} else {
			query += ", " + setValue
		}
	}

	// Add WHERE clause
	query += fmt.Sprintf(" WHERE id = $%d", paramCount)
	params = append(params, id)

	var execContext interface {
		ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	}

	if tx != nil {
		execContext = tx
	} else {
		execContext = r.db
	}

	_, err := execContext.ExecContext(ctx, query, params...)
	if err != nil {
		r.logger.Error("Failed to update strategy", zap.Error(err), zap.Int("id", id))
		return err
	}

	return nil
}

// Delete deletes a strategy
func (r *StrategyRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM strategies WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete strategy", zap.Error(err), zap.Int("id", id))
		return err
	}

	return nil
}

// getStrategyTags retrieves tags for a strategy
func (r *StrategyRepository) getStrategyTags(ctx context.Context, strategyID int) ([]model.Tag, error) {
	query := `
		SELECT t.id, t.name
		FROM strategy_tags t
		JOIN strategy_tag_mappings m ON t.id = m.tag_id
		WHERE m.strategy_id = $1
	`

	var tags []model.Tag
	err := r.db.SelectContext(ctx, &tags, query, strategyID)
	if err != nil {
		return nil, err
	}

	return tags, nil
}

// UpdateTags updates the tags for a strategy
func (r *StrategyRepository) UpdateTags(ctx context.Context, tx *sqlx.Tx, strategyID int, tagIDs []int) error {
	deleteQuery := `DELETE FROM strategy_tag_mappings WHERE strategy_id = $1`

	var execContext interface {
		ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
		QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	}

	if tx != nil {
		execContext = tx
	} else {
		execContext = r.db
	}

	// Delete existing mappings
	_, err := execContext.ExecContext(ctx, deleteQuery, strategyID)
	if err != nil {
		r.logger.Error("Failed to delete strategy tag mappings", zap.Error(err), zap.Int("strategy_id", strategyID))
		return err
	}

	// If no tags, we're done
	if len(tagIDs) == 0 {
		return nil
	}

	// Create new mappings
	insertQuery := `INSERT INTO strategy_tag_mappings (strategy_id, tag_id) VALUES `
	params := []interface{}{strategyID}
	paramIndex := 2

	for i, tagID := range tagIDs {
		if i > 0 {
			insertQuery += ", "
		}
		insertQuery += fmt.Sprintf("($1, $%d)", paramIndex)
		params = append(params, tagID)
		paramIndex++
	}

	_, err = execContext.ExecContext(ctx, insertQuery, params...)
	if err != nil {
		r.logger.Error("Failed to insert strategy tag mappings", zap.Error(err), zap.Int("strategy_id", strategyID))
		return err
	}

	return nil
}

// VersionRepository handles database operations for strategy versions
type VersionRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewVersionRepository creates a new version repository
func NewVersionRepository(db *sqlx.DB, logger *zap.Logger) *VersionRepository {
	return &VersionRepository{
		db:     db,
		logger: logger,
	}
}

// Create adds a new strategy version
func (r *VersionRepository) Create(ctx context.Context, tx *sqlx.Tx, version *model.VersionCreate, strategyID int, versionNumber int) (int, error) {
	query := `
		INSERT INTO strategy_versions (strategy_id, version, structure, change_notes, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	// Convert structure to JSON
	structureBytes, err := json.Marshal(version.Structure)
	if err != nil {
		r.logger.Error("Failed to marshal version structure", zap.Error(err))
		return 0, err
	}

	var id int
	var execContext interface {
		QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	}

	if tx != nil {
		execContext = tx
	} else {
		execContext = r.db
	}

	err = execContext.QueryRowContext(
		ctx,
		query,
		strategyID,
		versionNumber,
		structureBytes,
		version.ChangeNotes,
		time.Now(),
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to create strategy version", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// GetVersions retrieves all versions for a strategy
func (r *VersionRepository) GetVersions(ctx context.Context, strategyID int) ([]model.StrategyVersion, error) {
	query := `
		SELECT id, strategy_id, version, structure, change_notes, created_at
		FROM strategy_versions
		WHERE strategy_id = $1
		ORDER BY version DESC
	`

	rows, err := r.db.QueryContext(ctx, query, strategyID)
	if err != nil {
		r.logger.Error("Failed to query strategy versions", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	versions := []model.StrategyVersion{}
	for rows.Next() {
		var version model.StrategyVersion
		var structureBytes []byte

		err := rows.Scan(
			&version.ID,
			&version.StrategyID,
			&version.Version,
			&structureBytes,
			&version.ChangeNotes,
			&version.CreatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan version row", zap.Error(err))
			return nil, err
		}

		// Unmarshal the structure
		if err := json.Unmarshal(structureBytes, &version.Structure); err != nil {
			r.logger.Error("Failed to unmarshal version structure", zap.Error(err))
			return nil, err
		}

		versions = append(versions, version)
	}

	return versions, nil
}

// GetVersion retrieves a specific version of a strategy
func (r *VersionRepository) GetVersion(ctx context.Context, strategyID int, versionNumber int) (*model.StrategyVersion, error) {
	query := `
		SELECT id, strategy_id, version, structure, change_notes, created_at
		FROM strategy_versions
		WHERE strategy_id = $1 AND version = $2
	`

	var version model.StrategyVersion
	var structureBytes []byte

	row := r.db.QueryRowContext(ctx, query, strategyID, versionNumber)
	err := row.Scan(
		&version.ID,
		&version.StrategyID,
		&version.Version,
		&structureBytes,
		&version.ChangeNotes,
		&version.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get strategy version", zap.Error(err))
		return nil, err
	}

	// Unmarshal the structure
	if err := json.Unmarshal(structureBytes, &version.Structure); err != nil {
		r.logger.Error("Failed to unmarshal version structure", zap.Error(err))
		return nil, err
	}

	return &version, nil
}

// GetLatestVersion retrieves the latest version number for a strategy
func (r *VersionRepository) GetLatestVersion(ctx context.Context, strategyID int) (int, error) {
	query := `
		SELECT MAX(version)
		FROM strategy_versions
		WHERE strategy_id = $1
	`

	var version int
	err := r.db.GetContext(ctx, &version, query, strategyID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		r.logger.Error("Failed to get latest version", zap.Error(err))
		return 0, err
	}

	return version, nil
}

// TagRepository handles database operations for strategy tags
type TagRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewTagRepository creates a new tag repository
func NewTagRepository(db *sqlx.DB, logger *zap.Logger) *TagRepository {
	return &TagRepository{
		db:     db,
		logger: logger,
	}
}

// GetAll retrieves all tags
func (r *TagRepository) GetAll(ctx context.Context) ([]model.Tag, error) {
	query := `SELECT id, name FROM strategy_tags ORDER BY name`

	var tags []model.Tag
	err := r.db.SelectContext(ctx, &tags, query)
	if err != nil {
		r.logger.Error("Failed to get tags", zap.Error(err))
		return nil, err
	}

	return tags, nil
}

// GetByID retrieves a tag by ID
func (r *TagRepository) GetByID(ctx context.Context, id int) (*model.Tag, error) {
	query := `SELECT id, name FROM strategy_tags WHERE id = $1`

	var tag model.Tag
	err := r.db.GetContext(ctx, &tag, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get tag by ID", zap.Error(err))
		return nil, err
	}

	return &tag, nil
}

// Create adds a new tag
func (r *TagRepository) Create(ctx context.Context, name string) (int, error) {
	query := `INSERT INTO strategy_tags (name) VALUES ($1) RETURNING id`

	var id int
	err := r.db.QueryRowContext(ctx, query, name).Scan(&id)
	if err != nil {
		r.logger.Error("Failed to create tag", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// Update updates a tag
func (r *TagRepository) Update(ctx context.Context, id int, name string) error {
	query := `UPDATE strategy_tags SET name = $1 WHERE id = $2`

	_, err := r.db.ExecContext(ctx, query, name, id)
	if err != nil {
		r.logger.Error("Failed to update tag", zap.Error(err))
		return err
	}

	return nil
}

// Delete deletes a tag
func (r *TagRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM strategy_tags WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete tag", zap.Error(err))
		return err
	}

	return nil
}

// IndicatorRepository handles database operations for technical indicators
type IndicatorRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewIndicatorRepository creates a new indicator repository
func NewIndicatorRepository(db *sqlx.DB, logger *zap.Logger) *IndicatorRepository {
	return &IndicatorRepository{
		db:     db,
		logger: logger,
	}
}

// GetAll retrieves all indicators with optional category filter
func (r *IndicatorRepository) GetAll(ctx context.Context, category *string) ([]model.TechnicalIndicator, error) {
	query := `
		SELECT id, name, description, category, formula, created_at, updated_at
		FROM indicators
	`

	params := []interface{}{}
	if category != nil {
		query += " WHERE category = $1"
		params = append(params, *category)
	}

	query += " ORDER BY name"

	var indicators []model.TechnicalIndicator
	err := r.db.SelectContext(ctx, &indicators, query, params...)
	if err != nil {
		r.logger.Error("Failed to get indicators", zap.Error(err))
		return nil, err
	}

	// For each indicator, get its parameters
	for i := range indicators {
		parameters, err := r.getIndicatorParameters(ctx, indicators[i].ID)
		if err != nil {
			r.logger.Warn("Failed to get indicator parameters", zap.Error(err), zap.Int("indicator_id", indicators[i].ID))
		} else {
			indicators[i].Parameters = parameters
		}
	}

	return indicators, nil
}

// GetByID retrieves an indicator by ID
func (r *IndicatorRepository) GetByID(ctx context.Context, id int) (*model.TechnicalIndicator, error) {
	query := `
		SELECT id, name, description, category, formula, created_at, updated_at
		FROM indicators
		WHERE id = $1
	`

	var indicator model.TechnicalIndicator
	err := r.db.GetContext(ctx, &indicator, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get indicator by ID", zap.Error(err))
		return nil, err
	}

	// Get parameters for the indicator
	parameters, err := r.getIndicatorParameters(ctx, id)
	if err != nil {
		r.logger.Warn("Failed to get indicator parameters", zap.Error(err), zap.Int("indicator_id", id))
	} else {
		indicator.Parameters = parameters
	}

	return &indicator, nil
}

// getIndicatorParameters retrieves parameters for an indicator
func (r *IndicatorRepository) getIndicatorParameters(ctx context.Context, indicatorID int) ([]model.IndicatorParameter, error) {
	query := `
		SELECT id, indicator_id, parameter_name, parameter_type, is_required, min_value, max_value, default_value, description
		FROM indicator_parameters
		WHERE indicator_id = $1
		ORDER BY id
	`

	var parameters []model.IndicatorParameter
	err := r.db.SelectContext(ctx, &parameters, query, indicatorID)
	if err != nil {
		return nil, err
	}

	// For each parameter, get enum values if applicable
	for i, param := range parameters {
		if param.ParameterType == "enum" {
			enumValues, err := r.getParameterEnumValues(ctx, param.ID)
			if err != nil {
				r.logger.Warn("Failed to get parameter enum values", zap.Error(err), zap.Int("parameter_id", param.ID))
			} else {
				parameters[i].EnumValues = enumValues
			}
		}
	}

	return parameters, nil
}

// getParameterEnumValues retrieves enum values for a parameter
func (r *IndicatorRepository) getParameterEnumValues(ctx context.Context, parameterID int) ([]model.ParameterEnumValue, error) {
	query := `
		SELECT id, parameter_id, enum_value, display_name
		FROM parameter_enum_values
		WHERE parameter_id = $1
		ORDER BY id
	`

	var enumValues []model.ParameterEnumValue
	err := r.db.SelectContext(ctx, &enumValues, query, parameterID)
	if err != nil {
		return nil, err
	}

	return enumValues, nil
}

// MarketplaceRepository handles database operations for the strategy marketplace
type MarketplaceRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewMarketplaceRepository creates a new marketplace repository
func NewMarketplaceRepository(db *sqlx.DB, logger *zap.Logger) *MarketplaceRepository {
	return &MarketplaceRepository{
		db:     db,
		logger: logger,
	}
}

// Create adds a new marketplace listing
func (r *MarketplaceRepository) Create(ctx context.Context, listing *model.MarketplaceCreate, userID int) (int, error) {
	query := `
		INSERT INTO strategy_marketplace (strategy_id, user_id, price, is_subscription, subscription_period, is_active, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		listing.StrategyID,
		userID,
		listing.Price,
		listing.IsSubscription,
		listing.SubscriptionPeriod,
		true, // Active by default
		listing.Description,
		time.Now(),
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to create marketplace listing", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// GetByID retrieves a marketplace listing by ID
func (r *MarketplaceRepository) GetByID(ctx context.Context, id int) (*model.MarketplaceItem, error) {
	query := `
		SELECT id, strategy_id, user_id, price, is_subscription, subscription_period, is_active, description, created_at, updated_at
		FROM strategy_marketplace
		WHERE id = $1
	`

	var item model.MarketplaceItem
	err := r.db.GetContext(ctx, &item, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get marketplace item", zap.Error(err))
		return nil, err
	}

	return &item, nil
}

// GetAll retrieves marketplace listings with filters and pagination
func (r *MarketplaceRepository) GetAll(ctx context.Context, isActive *bool, userID *int, page, limit int) ([]model.MarketplaceItem, int, error) {
	query := `
		SELECT m.id, m.strategy_id, m.user_id, m.price, m.is_subscription, 
		       m.subscription_period, m.is_active, m.description, m.created_at, m.updated_at
		FROM strategy_marketplace m
		WHERE 1=1
	`

	countQuery := `
		SELECT COUNT(*)
		FROM strategy_marketplace m
		WHERE 1=1
	`

	params := []interface{}{}
	paramIndex := 1

	// Add filters
	if isActive != nil {
		query += fmt.Sprintf(" AND m.is_active = $%d", paramIndex)
		countQuery += fmt.Sprintf(" AND m.is_active = $%d", paramIndex)
		params = append(params, *isActive)
		paramIndex++
	}

	if userID != nil {
		query += fmt.Sprintf(" AND m.user_id = $%d", paramIndex)
		countQuery += fmt.Sprintf(" AND m.user_id = $%d", paramIndex)
		params = append(params, *userID)
		paramIndex++
	}

	// Get total count
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, params...)
	if err != nil {
		r.logger.Error("Failed to count marketplace items", zap.Error(err))
		return nil, 0, err
	}

	// Add pagination
	offset := (page - 1) * limit
	query += fmt.Sprintf(" ORDER BY m.created_at DESC LIMIT $%d OFFSET $%d", paramIndex, paramIndex+1)
	params = append(params, limit, offset)

	// Execute query
	var items []model.MarketplaceItem
	err = r.db.SelectContext(ctx, &items, query, params...)
	if err != nil {
		r.logger.Error("Failed to get marketplace items", zap.Error(err))
		return nil, 0, err
	}

	return items, total, nil
}

// Update updates a marketplace listing
func (r *MarketplaceRepository) Update(ctx context.Context, id int, price *float64, isActive *bool, description *string) error {
	query := `
		UPDATE strategy_marketplace
		SET
	`

	params := []interface{}{}
	paramCount := 1
	setValues := []string{}

	if price != nil {
		setValues = append(setValues, fmt.Sprintf("price = $%d", paramCount))
		params = append(params, *price)
		paramCount++
	}

	if isActive != nil {
		setValues = append(setValues, fmt.Sprintf("is_active = $%d", paramCount))
		params = append(params, *isActive)
		paramCount++
	}

	if description != nil {
		setValues = append(setValues, fmt.Sprintf("description = $%d", paramCount))
		params = append(params, *description)
		paramCount++
	}

	// Always update the updated_at timestamp
	setValues = append(setValues, fmt.Sprintf("updated_at = $%d", paramCount))
	params = append(params, time.Now())
	paramCount++

	// If no fields were provided for update, return
	if len(setValues) == 1 { // Only updated_at
		return nil
	}

	// Combine SET clauses
	for i, setValue := range setValues {
		if i == 0 {
			query += setValue
		} else {
			query += ", " + setValue
		}
	}

	// Add WHERE clause
	query += fmt.Sprintf(" WHERE id = $%d", paramCount)
	params = append(params, id)

	_, err := r.db.ExecContext(ctx, query, params...)
	if err != nil {
		r.logger.Error("Failed to update marketplace item", zap.Error(err))
		return err
	}

	return nil
}

// Delete removes a marketplace listing
func (r *MarketplaceRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM strategy_marketplace WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete marketplace item", zap.Error(err))
		return err
	}

	return nil
}

// PurchaseRepository handles database operations for strategy purchases
type PurchaseRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewPurchaseRepository creates a new purchase repository
func NewPurchaseRepository(db *sqlx.DB, logger *zap.Logger) *PurchaseRepository {
	return &PurchaseRepository{
		db:     db,
		logger: logger,
	}
}

// Create adds a new purchase record
func (r *PurchaseRepository) Create(ctx context.Context, marketplaceID int, buyerID int, price float64, subscriptionEnd *time.Time) (int, error) {
	query := `
		INSERT INTO strategy_purchases (marketplace_id, buyer_id, purchase_price, subscription_end, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		marketplaceID,
		buyerID,
		price,
		subscriptionEnd,
		time.Now(),
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to create purchase", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// GetByUser retrieves purchases for a user
func (r *PurchaseRepository) GetByUser(ctx context.Context, userID int, page, limit int) ([]model.StrategyPurchase, int, error) {
	query := `
		SELECT id, marketplace_id, buyer_id, purchase_price, subscription_end, created_at
		FROM strategy_purchases
		WHERE buyer_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	countQuery := `
		SELECT COUNT(*)
		FROM strategy_purchases
		WHERE buyer_id = $1
	`

	// Get total count
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, userID)
	if err != nil {
		r.logger.Error("Failed to count purchases", zap.Error(err))
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Execute main query
	var purchases []model.StrategyPurchase
	err = r.db.SelectContext(ctx, &purchases, query, userID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to get purchases", zap.Error(err))
		return nil, 0, err
	}

	return purchases, total, nil
}

// HasPurchased checks if a user has purchased a specific strategy
func (r *PurchaseRepository) HasPurchased(ctx context.Context, userID int, strategyID int) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM strategy_purchases p
		JOIN strategy_marketplace m ON p.marketplace_id = m.id
		WHERE p.buyer_id = $1 AND m.strategy_id = $2
		AND (p.subscription_end IS NULL OR p.subscription_end > NOW())
	`

	var count int
	err := r.db.GetContext(ctx, &count, query, userID, strategyID)
	if err != nil {
		r.logger.Error("Failed to check purchase status", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}

// ReviewRepository handles database operations for strategy reviews
type ReviewRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewReviewRepository creates a new review repository
func NewReviewRepository(db *sqlx.DB, logger *zap.Logger) *ReviewRepository {
	return &ReviewRepository{
		db:     db,
		logger: logger,
	}
}

// Create adds a new review
func (r *ReviewRepository) Create(ctx context.Context, review *model.ReviewCreate, userID int) (int, error) {
	query := `
		INSERT INTO strategy_reviews (marketplace_id, user_id, rating, comment, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		review.MarketplaceID,
		userID,
		review.Rating,
		review.Comment,
		time.Now(),
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to create review", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// Update updates a review
func (r *ReviewRepository) Update(ctx context.Context, id int, rating int, comment string) error {
	query := `
		UPDATE strategy_reviews
		SET rating = $1, comment = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		rating,
		comment,
		time.Now(),
		id,
	)

	if err != nil {
		r.logger.Error("Failed to update review", zap.Error(err))
		return err
	}

	return nil
}

// Delete deletes a review
func (r *ReviewRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM strategy_reviews WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete review", zap.Error(err))
		return err
	}

	return nil
}

// GetByMarketplaceID retrieves reviews for a marketplace listing
func (r *ReviewRepository) GetByMarketplaceID(ctx context.Context, marketplaceID int, page, limit int) ([]model.StrategyReview, int, error) {
	query := `
		SELECT id, marketplace_id, user_id, rating, comment, created_at, updated_at
		FROM strategy_reviews
		WHERE marketplace_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	countQuery := `
		SELECT COUNT(*)
		FROM strategy_reviews
		WHERE marketplace_id = $1
	`

	// Get total count
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, marketplaceID)
	if err != nil {
		r.logger.Error("Failed to count reviews", zap.Error(err))
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Execute main query
	var reviews []model.StrategyReview
	err = r.db.SelectContext(ctx, &reviews, query, marketplaceID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to get reviews", zap.Error(err))
		return nil, 0, err
	}

	return reviews, total, nil
}

// GetAverageRating calculates the average rating for a marketplace listing
func (r *ReviewRepository) GetAverageRating(ctx context.Context, marketplaceID int) (float64, error) {
	query := `
		SELECT COALESCE(AVG(rating), 0) 
		FROM strategy_reviews
		WHERE marketplace_id = $1
	`

	var avgRating float64
	err := r.db.GetContext(ctx, &avgRating, query, marketplaceID)
	if err != nil {
		r.logger.Error("Failed to get average rating", zap.Error(err))
		return 0, err
	}

	return avgRating, nil
}

// HasReviewed checks if a user has already reviewed a marketplace listing
func (r *ReviewRepository) HasReviewed(ctx context.Context, userID int, marketplaceID int) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM strategy_reviews
		WHERE user_id = $1 AND marketplace_id = $2
	`

	var count int
	err := r.db.GetContext(ctx, &count, query, userID, marketplaceID)
	if err != nil {
		r.logger.Error("Failed to check review status", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}
