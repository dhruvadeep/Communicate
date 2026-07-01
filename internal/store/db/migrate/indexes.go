package migrate

import (
	"context"
	"fmt"

	"Communicate/internal/store/db/models/structure"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunIndexes creates all indexes defined in the provided Index structs.
func RunIndexes(ctx context.Context, pool *pgxpool.Pool, indexes ...structure.Index) error {
	for _, idx := range indexes {
		for _, cmd := range idx.CreateSQL {
			if _, err := pool.Exec(ctx, cmd); err != nil {
				return fmt.Errorf("create index for %s: %w", idx.TableName, err)
			}
		}
	}
	return nil
}

// DropIndexes drops all indexes defined in the provided Index structs.
func DropIndexes(ctx context.Context, pool *pgxpool.Pool, indexes ...structure.Index) error {
	for _, idx := range indexes {
		for _, cmd := range idx.DropSQL {
			if _, err := pool.Exec(ctx, cmd); err != nil {
				return fmt.Errorf("drop index for %s: %w", idx.TableName, err)
			}
		}
	}
	return nil
}
