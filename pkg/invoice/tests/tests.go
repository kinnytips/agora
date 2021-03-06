package tests

import (
	"context"
	"crypto/sha256"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonpb "github.com/kinecosystem/agora-api/genproto/common/v3"

	"github.com/kinecosystem/agora/pkg/invoice"
)

func RunTests(t *testing.T, store invoice.Store, teardown func()) {
	for _, tf := range []func(*testing.T, invoice.Store){testRoundTrip, testExists} {
		tf(t, store)
		teardown()
	}
}

func testRoundTrip(t *testing.T, store invoice.Store) {
	t.Run("TestRoundTrip", func(t *testing.T) {
		h := sha256.Sum256([]byte("somedata"))
		txHash := h[:]

		il := &commonpb.InvoiceList{
			Invoices: []*commonpb.Invoice{
				{
					Items: []*commonpb.Invoice_LineItem{
						{
							Title:       "lineitem1",
							Description: "desc1",
							Amount:      5,
							Sku:         nil,
						},
						{
							Title:       "lineitem2",
							Description: "desc3",
							Amount:      2,
							Sku:         nil,
						},
					},
				},
			},
		}

		// Doesn't exist yet
		record, err := store.Get(context.Background(), txHash)
		require.Equal(t, invoice.ErrNotFound, err)
		require.Nil(t, record)

		require.NoError(t, store.Put(context.Background(), txHash, il))

		actual, err := store.Get(context.Background(), txHash)
		require.NoError(t, err)
		require.True(t, proto.Equal(il, actual))

		// Ensure non-32 byte txHashs cannot be used.
		err = store.Put(context.Background(), []byte{1, 2, 3}, il)
		assert.NotNil(t, err)
		assert.NotEqual(t, invoice.ErrNotFound, err)

		_, err = store.Get(context.Background(), []byte{1, 2, 3})
		assert.NotNil(t, err)
		assert.NotEqual(t, invoice.ErrNotFound, err)
	})
}

func testExists(t *testing.T, store invoice.Store) {
	t.Run("TestCollision", func(t *testing.T) {
		h := sha256.Sum256([]byte("somedata"))
		txHash := h[:]

		il := &commonpb.InvoiceList{
			Invoices: []*commonpb.Invoice{
				{
					Items: []*commonpb.Invoice_LineItem{
						{
							Title:       "lineitem1",
							Description: "desc1",
							Amount:      5,
							Sku:         nil,
						},
						{
							Title:       "lineitem2",
							Description: "desc3",
							Amount:      2,
							Sku:         nil,
						},
					},
				},
			},
		}

		require.NoError(t, store.Put(context.Background(), txHash, il))

		err := store.Put(context.Background(), txHash, il)
		require.Equal(t, invoice.ErrExists, err)
	})
}
