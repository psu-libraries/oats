package oabutton_test

import (
	"testing"

	"git.psu.edu/sre53/oats/oabutton"
	"github.com/matryer/is"
)

func TestPermissions(t *testing.T) {
	is := is.New(t)
	c := oabutton.NewClient("")
	perms, err := c.GetPermissions("10.1037/apl0000872")
	is.NoErr(err)
	is.True(len(perms) > 0)
	is.Equal(perms[0].ScholarSphereOK(), true)
	is.Equal(perms[0].BestLicense(), "other-closed")

	err = c.TestPermissionsAPI()
	is.NoErr(err)
}
