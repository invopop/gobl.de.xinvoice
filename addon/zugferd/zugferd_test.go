package zugferd_test

import (
	"testing"

	"github.com/invopop/gobl.de.xinvoice/addon/zugferd"
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddon(t *testing.T) {
	ad := tax.AddonForKey(zugferd.V2)
	require.NotNil(t, ad)
	assert.Equal(t, "German ZUGFeRD 2.X", ad.Name.String())
	assert.Equal(t, en16931.V2017, ad.Requires[0])
}
