package repository

import (
	"fmt"
	"strings"

	"github.com/souvik03-136/Go-Store/internal/models"
)

// SearchOwned applies owner-scoped filters; dynamic identifiers are allowlisted.
func (r *FileRepository) SearchOwned(ownerID, query, contentType, sortBy, direction string, folderID *string, limit, offset int) ([]*models.File, error) {
	columns := map[string]string{"name": "name", "size": "size", "created_at": "created_at", "updated_at": "updated_at"}
	column, ok := columns[sortBy]
	if !ok {
		column = "updated_at"
	}
	direction = strings.ToUpper(direction)
	if direction != "ASC" {
		direction = "DESC"
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	args := []any{ownerID}
	where := []string{"owner_id=$1", "state='active'"}
	add := func(v any) string { args = append(args, v); return fmt.Sprintf("$%d", len(args)) }
	if query != "" {
		p := add("%" + query + "%")
		where = append(where, "name ILIKE "+p)
	}
	if contentType != "" {
		p := add(contentType + "%")
		where = append(where, "content_type ILIKE "+p)
	}
	if folderID != nil {
		if *folderID == "" {
			where = append(where, "folder_id IS NULL")
		} else {
			where = append(where, "folder_id="+add(*folderID))
		}
	}
	limitParam := add(limit)
	offsetParam := add(offset)
	querySQL := fmt.Sprintf(`SELECT %s FROM files WHERE %s ORDER BY %s %s, id ASC LIMIT %s OFFSET %s`, fileColumns, strings.Join(where, " AND "), column, direction, limitParam, offsetParam)
	rows, err := r.db.Query(querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("searching files: %w", err)
	}
	defer rows.Close()
	files := make([]*models.File, 0)
	for rows.Next() {
		f, err := scanFileRows(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating search results: %w", err)
	}
	return files, nil
}
