package whatsmeow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mau.fi/whatsmeow/types"
)

func TestLidPhoneCandidates(t *testing.T) {
	lid := "269638281724102"
	full := lid + "@" + string(types.HiddenUserServer)

	candidates := lidPhoneCandidates(lid)
	assert.ElementsMatch(t, []string{lid, full}, candidates)

	candidates = lidPhoneCandidates(full)
	assert.ElementsMatch(t, []string{lid, full}, candidates)
}
