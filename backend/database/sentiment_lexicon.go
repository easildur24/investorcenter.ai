package database

import (
	"database/sql"
	"fmt"
	"investorcenter-api/models"
	"strings"
)

// GetSentimentLexicon returns all terms in the sentiment lexicon
func GetSentimentLexicon() ([]models.SentimentLexiconTerm, error) {
	query := `
		SELECT id, term, sentiment, weight, category, created_at
		FROM sentiment_lexicon
		ORDER BY sentiment, weight DESC
	`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sentiment lexicon: %w", err)
	}
	defer rows.Close()

	var terms []models.SentimentLexiconTerm
	for rows.Next() {
		var t models.SentimentLexiconTerm
		var category sql.NullString

		err := rows.Scan(
			&t.ID,
			&t.Term,
			&t.Sentiment,
			&t.Weight,
			&category,
			&t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sentiment term: %w", err)
		}

		if category.Valid {
			t.Category = &category.String
		}

		terms = append(terms, t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sentiment terms: %w", err)
	}

	return terms, nil
}

// GetSentimentTermsBySentiment returns terms filtered by sentiment type
func GetSentimentTermsBySentiment(sentiment string) ([]models.SentimentLexiconTerm, error) {
	query := `
		SELECT id, term, sentiment, weight, category, created_at
		FROM sentiment_lexicon
		WHERE sentiment = $1
		ORDER BY weight DESC
	`

	rows, err := DB.Query(query, sentiment)
	if err != nil {
		return nil, fmt.Errorf("failed to query sentiment terms by type: %w", err)
	}
	defer rows.Close()

	var terms []models.SentimentLexiconTerm
	for rows.Next() {
		var t models.SentimentLexiconTerm
		var category sql.NullString

		err := rows.Scan(
			&t.ID,
			&t.Term,
			&t.Sentiment,
			&t.Weight,
			&category,
			&t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sentiment term: %w", err)
		}

		if category.Valid {
			t.Category = &category.String
		}

		terms = append(terms, t)
	}

	return terms, nil
}

// GetSentimentTermsByCategory returns terms filtered by category
func GetSentimentTermsByCategory(category string) ([]models.SentimentLexiconTerm, error) {
	query := `
		SELECT id, term, sentiment, weight, category, created_at
		FROM sentiment_lexicon
		WHERE category = $1
		ORDER BY sentiment, weight DESC
	`

	rows, err := DB.Query(query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to query sentiment terms by category: %w", err)
	}
	defer rows.Close()

	var terms []models.SentimentLexiconTerm
	for rows.Next() {
		var t models.SentimentLexiconTerm
		var cat sql.NullString

		err := rows.Scan(
			&t.ID,
			&t.Term,
			&t.Sentiment,
			&t.Weight,
			&cat,
			&t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sentiment term: %w", err)
		}

		if cat.Valid {
			t.Category = &cat.String
		}

		terms = append(terms, t)
	}

	return terms, nil
}

// AddSentimentTerm adds a new term to the lexicon
func AddSentimentTerm(term *models.SentimentLexiconTerm) error {
	query := `
		INSERT INTO sentiment_lexicon (term, sentiment, weight, category)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (term) DO UPDATE SET
			sentiment = EXCLUDED.sentiment,
			weight = EXCLUDED.weight,
			category = EXCLUDED.category
		RETURNING id, created_at
	`

	err := DB.QueryRow(query,
		strings.ToLower(term.Term),
		term.Sentiment,
		term.Weight,
		term.Category,
	).Scan(&term.ID, &term.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to add sentiment term: %w", err)
	}
	return nil
}

// DeleteSentimentTerm removes a term from the lexicon
func DeleteSentimentTerm(termID int) error {
	result, err := DB.Exec("DELETE FROM sentiment_lexicon WHERE id = $1", termID)
	if err != nil {
		return fmt.Errorf("failed to delete sentiment term: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("term not found")
	}

	return nil
}

// LookupTerm finds a term in the lexicon (case-insensitive)
func LookupTerm(term string) (*models.SentimentLexiconTerm, error) {
	query := `
		SELECT id, term, sentiment, weight, category, created_at
		FROM sentiment_lexicon
		WHERE LOWER(term) = LOWER($1)
	`

	var t models.SentimentLexiconTerm
	var category sql.NullString

	err := DB.QueryRow(query, term).Scan(
		&t.ID,
		&t.Term,
		&t.Sentiment,
		&t.Weight,
		&category,
		&t.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Term not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to lookup term: %w", err)
	}

	if category.Valid {
		t.Category = &category.String
	}

	return &t, nil
}

// GetLexiconAsMap returns the lexicon as a map for efficient lookups
// Map key is lowercase term, value is the full term data
func GetLexiconAsMap() (map[string]models.SentimentLexiconTerm, error) {
	terms, err := GetSentimentLexicon()
	if err != nil {
		return nil, err
	}

	lexiconMap := make(map[string]models.SentimentLexiconTerm, len(terms))
	for _, term := range terms {
		lexiconMap[strings.ToLower(term.Term)] = term
	}

	return lexiconMap, nil
}

// GetLexiconStats returns statistics about the lexicon
func GetLexiconStats() (map[string]int, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE sentiment = 'bullish') as bullish,
			COUNT(*) FILTER (WHERE sentiment = 'bearish') as bearish,
			COUNT(*) FILTER (WHERE sentiment = 'modifier') as modifiers
		FROM sentiment_lexicon
	`

	var total, bullish, bearish, modifiers int
	err := DB.QueryRow(query).Scan(&total, &bullish, &bearish, &modifiers)
	if err != nil {
		return nil, fmt.Errorf("failed to get lexicon stats: %w", err)
	}

	return map[string]int{
		"total":     total,
		"bullish":   bullish,
		"bearish":   bearish,
		"modifiers": modifiers,
	}, nil
}

// Social Data Sources

// GetEnabledDataSources returns all enabled social data sources
func GetEnabledDataSources() ([]models.SocialDataSource, error) {
	query := `
		SELECT id, source_name, is_enabled, config, created_at, updated_at
		FROM social_data_sources
		WHERE is_enabled = true
		ORDER BY source_name
	`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query data sources: %w", err)
	}
	defer rows.Close()

	var sources []models.SocialDataSource
	for rows.Next() {
		var s models.SocialDataSource
		var configBytes []byte

		err := rows.Scan(
			&s.ID,
			&s.SourceName,
			&s.IsEnabled,
			&configBytes,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan data source: %w", err)
		}

		// Parse JSON config - for now just store raw bytes
		// Full JSONB parsing can be added later
		sources = append(sources, s)
	}

	return sources, nil
}

// GetDataSourceByName returns a specific data source by name
func GetDataSourceByName(name string) (*models.SocialDataSource, error) {
	query := `
		SELECT id, source_name, is_enabled, config, created_at, updated_at
		FROM social_data_sources
		WHERE source_name = $1
	`

	var s models.SocialDataSource
	var configBytes []byte

	err := DB.QueryRow(query, name).Scan(
		&s.ID,
		&s.SourceName,
		&s.IsEnabled,
		&configBytes,
		&s.CreatedAt,
		&s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get data source: %w", err)
	}

	return &s, nil
}

// UpdateDataSourceEnabled enables or disables a data source
func UpdateDataSourceEnabled(name string, enabled bool) error {
	query := `
		UPDATE social_data_sources
		SET is_enabled = $1, updated_at = NOW()
		WHERE source_name = $2
	`

	result, err := DB.Exec(query, enabled, name)
	if err != nil {
		return fmt.Errorf("failed to update data source: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("data source not found: %s", name)
	}

	return nil
}
