package application

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarket_createInlineKeyboardForPlanets(t *testing.T) {
	ids := []int{234, 432, 3423, 4324, 234}
	k := createInlineKeyboardForPlanets(ids)
	require.Equal(t, 2, len(k.InlineKeyboard))
}
