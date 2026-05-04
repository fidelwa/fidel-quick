package phone

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func sorted(s []string) []string {
	sort.Strings(s)
	return s
}

func TestVariants_MexicanMobileWith521(t *testing.T) {
	// WhatsApp sends this format
	v := sorted(Variants("5215573911640"))
	assert.Contains(t, v, "5215573911640")  // original
	assert.Contains(t, v, "525573911640")   // without 1
	assert.Contains(t, v, "5573911640")     // 10 digits
	assert.Contains(t, v, "+5215573911640") // with +
}

func TestVariants_MexicanWithout1(t *testing.T) {
	v := sorted(Variants("525573911640"))
	assert.Contains(t, v, "525573911640")   // original
	assert.Contains(t, v, "5215573911640")  // with 1
	assert.Contains(t, v, "5573911640")     // 10 digits
}

func TestVariants_10Digits(t *testing.T) {
	v := sorted(Variants("5573911640"))
	assert.Contains(t, v, "5573911640")    // original
	assert.Contains(t, v, "525573911640")  // with 52
	assert.Contains(t, v, "5215573911640") // with 521
}

func TestVariants_11DigitsWithPrefix1(t *testing.T) {
	// Old format: 1 + 10 digits
	v := sorted(Variants("15573911640"))
	assert.Contains(t, v, "15573911640")   // original
	assert.Contains(t, v, "5215573911640") // 52 + 1 + 10
	assert.Contains(t, v, "525573911640")  // 52 + 10
	assert.Contains(t, v, "5573911640")    // 10 digits
}

func TestVariants_WithPlusPrefix(t *testing.T) {
	v := sorted(Variants("+5215573911640"))
	assert.Contains(t, v, "5215573911640")
	assert.Contains(t, v, "525573911640")
	assert.Contains(t, v, "5573911640")
}

func TestVariants_Empty(t *testing.T) {
	assert.Nil(t, Variants(""))
	assert.Nil(t, Variants("+"))
}

func TestVariants_NonMexican(t *testing.T) {
	// US number — should just return as-is
	v := Variants("12025551234")
	assert.Contains(t, v, "12025551234")
}

func TestVariants_CrossMatch_521vs10(t *testing.T) {
	// WhatsApp sends 521+10, DB stores 10 digits
	wa := Variants("5215514848559")
	db := Variants("5514848559")

	assert.True(t, hasOverlap(wa, db), "521+10 and 10-digit must share a variant")
}

func TestVariants_CrossMatch_521vs52(t *testing.T) {
	// WhatsApp sends 521+10, DB stores 52+10
	wa := Variants("5215514848559")
	db := Variants("525514848559")

	assert.True(t, hasOverlap(wa, db), "521+10 and 52+10 must share a variant")
}

func TestVariants_CrossMatch_52vs10(t *testing.T) {
	wa := Variants("525514848559")
	db := Variants("5514848559")

	assert.True(t, hasOverlap(wa, db), "52+10 and 10-digit must share a variant")
}

func TestVariants_CrossMatch_WithPlus(t *testing.T) {
	wa := Variants("+5215514848559")
	db := Variants("5514848559")

	assert.True(t, hasOverlap(wa, db), "+521 and 10-digit must share a variant")
}

func hasOverlap(a, b []string) bool {
	set := make(map[string]bool, len(a))
	for _, v := range a {
		set[v] = true
	}
	for _, v := range b {
		if set[v] {
			return true
		}
	}
	return false
}
