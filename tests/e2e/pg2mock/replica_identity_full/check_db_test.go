package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/transferia/transferia/internal/logger"
	"github.com/transferia/transferia/pkg/abstract"
	"github.com/transferia/transferia/pkg/abstract/model"
	"github.com/transferia/transferia/pkg/providers/postgres"
	"github.com/transferia/transferia/pkg/providers/postgres/pgrecipe"
	"github.com/transferia/transferia/tests/helpers"
)

var (
	Source = pgrecipe.RecipeSource(pgrecipe.WithPrefix(""), pgrecipe.WithInitDir("init_source"))
)

func init() {
	_ = os.Setenv("YC", "1") // to not go to vanga
	Source.WithDefaults()
}

//---------------------------------------------------------------------------------------------------------------------

func TestSnapshotAndIncrement(t *testing.T) {
	defer require.NoError(t, helpers.CheckConnections(
		helpers.LabeledPort{Label: "PG source", Port: Source.Port},
	))

	//------------------------------------------------------------------------------

	sinker := &helpers.MockSink{}
	target := model.MockDestination{
		SinkerFactory: func() abstract.Sinker { return sinker },
		Cleanup:       model.DisabledCleanup,
	}
	transfer := helpers.MakeTransfer("fake", Source, &target, abstract.TransferTypeSnapshotAndIncrement)
	checksTriggered := 0

	sinker.PushCallback = func(input []abstract.ChangeItem) error {
		for _, changeItem := range input {
			if changeItem.Kind == abstract.DeleteKind || changeItem.Kind == abstract.UpdateKind {
				checksTriggered += 1
				for _, col := range changeItem.TableSchema.Columns() {
					if col.PrimaryKey && col.FakeKey {
						require.Contains(t, changeItem.OldKeys.KeyNames, col.ColumnName)
					}
				}
			}
		}
		return nil
	}

	worker := helpers.Activate(t, transfer)
	defer worker.Close(t)

	ctx := context.Background()
	srcConn, err := postgres.MakeConnPoolFromSrc(Source, logger.Log)
	require.NoError(t, err)

	_, err = srcConn.Exec(ctx, `UPDATE public.test set another ='23' WHERE value = '11'`)
	require.NoError(t, err)

	_, err = srcConn.Exec(ctx, `DELETE FROM public.test  WHERE value = '21'`)
	require.NoError(t, err)

	require.NoError(t, helpers.WaitCond(time.Second*60, func() bool { return checksTriggered == 2 }))
}
