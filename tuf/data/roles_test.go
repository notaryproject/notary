package data

import (
	"path"
	"strings"
	"testing"

	"fmt"
	"github.com/stretchr/testify/require"
)

func TestMergeStrSlicesExclusive(t *testing.T) {
	orig := []string{"a"}
	new := []string{"b"}

	res := mergeStrSlices(orig, new)
	require.Len(t, res, 2)
	require.Equal(t, "a", res[0])
	require.Equal(t, "b", res[1])
}

func TestMergeStrSlicesOverlap(t *testing.T) {
	orig := []string{"a"}
	new := []string{"a", "b"}

	res := mergeStrSlices(orig, new)
	require.Len(t, res, 2)
	require.Equal(t, "a", res[0])
	require.Equal(t, "b", res[1])
}

func TestMergeStrSlicesEqual(t *testing.T) {
	orig := []string{"a"}
	new := []string{"a"}

	res := mergeStrSlices(orig, new)
	require.Len(t, res, 1)
	require.Equal(t, "a", res[0])
}

func TestSubtractStrSlicesExclusive(t *testing.T) {
	orig := []string{"a"}
	new := []string{"b"}

	res := subtractStrSlices(orig, new)
	require.Len(t, res, 1)
	require.Equal(t, "a", res[0])
}

func TestSubtractStrSlicesOverlap(t *testing.T) {
	orig := []string{"a", "b"}
	new := []string{"a"}

	res := subtractStrSlices(orig, new)
	require.Len(t, res, 1)
	require.Equal(t, "b", res[0])
}

func TestSubtractStrSlicesEqual(t *testing.T) {
	orig := []string{"a"}
	new := []string{"a"}

	res := subtractStrSlices(orig, new)
	require.Len(t, res, 0)
}

func TestAddRemoveKeys(t *testing.T) {
	role, err := NewRole("targets", 1, []string{"abc"}, []string{""})
	require.NoError(t, err)
	role.AddKeys([]string{"abc"})
	require.Equal(t, []string{"abc"}, role.KeyIDs)
	role.AddKeys([]string{"def"})
	require.Equal(t, []string{"abc", "def"}, role.KeyIDs)
	role.RemoveKeys([]string{"abc"})
	require.Equal(t, []string{"def"}, role.KeyIDs)
}

func TestAddRemovePaths(t *testing.T) {
	role, err := NewRole("targets", 1, []string{"abc"}, []string{"123"})
	require.NoError(t, err)
	err = role.AddPaths([]string{"123"})
	require.NoError(t, err)
	require.Equal(t, []string{"123"}, role.Paths)
	err = role.AddPaths([]string{"456"})
	require.NoError(t, err)
	require.Equal(t, []string{"123", "456"}, role.Paths)
	role.RemovePaths([]string{"123"})
	require.Equal(t, []string{"456"}, role.Paths)
}

func TestAddPathNil(t *testing.T) {
	role, err := NewRole("targets", 1, []string{"abc"}, nil)
	require.NoError(t, err)
	err = role.AddPaths(nil)
	require.NoError(t, err)
}

func TestErrNoSuchRole(t *testing.T) {
	var err error = ErrNoSuchRole{Role: "test"}
	require.True(t, strings.HasSuffix(err.Error(), "test"))
}

func TestErrInvalidRole(t *testing.T) {
	var err error = ErrInvalidRole{Role: "test"}
	require.False(t, strings.Contains(err.Error(), "Reason"))
}

func TestIsDelegation(t *testing.T) {
	f := require.False
	tr := require.True
	for val, check := range map[string]func(require.TestingT, bool, ...interface{}){
		// false checks
		path.Join(CanonicalTargetsRole, strings.Repeat("x", 255-len(CanonicalTargetsRole))): f,
		"":                                                                                  f,
		CanonicalRootRole:                                                                   f,
		path.Join(CanonicalRootRole, "level1"):                                              f,
		CanonicalTargetsRole:                                                                f,
		CanonicalTargetsRole + "/":                                                          f,
		path.Join(CanonicalTargetsRole, "level1") + "/":                                     f,
		path.Join(CanonicalTargetsRole, "UpperCase"):                                        f,
		path.Join(CanonicalTargetsRole, "directory") + "/../../traversal":                   f,
		CanonicalTargetsRole + "///test/middle/slashes":                                     f,
		CanonicalTargetsRole + "/./././":                                                    f,
		path.Join("  ", CanonicalTargetsRole, "level1"):                                     f,
		path.Join("  "+CanonicalTargetsRole, "level1"):                                      f,
		path.Join(CanonicalTargetsRole, "level1"+"  "):                                      f,
		path.Join(CanonicalTargetsRole, "white   space"+"level2"):                           f,
		path.Join(CanonicalTargetsRole, strings.Repeat("x", 256-len(CanonicalTargetsRole))): f,

		// true checks
		path.Join(CanonicalTargetsRole, "level1"):                     tr,
		path.Join(CanonicalTargetsRole, "level1", "level2", "level3"): tr,
		path.Join(CanonicalTargetsRole, "under_score"):                tr,
		path.Join(CanonicalTargetsRole, "hyphen-hyphen"):              tr,
	} {
		check(t, IsDelegation(val))
	}

}

func TestIsWildDelegation(t *testing.T) {
	f := require.False
	tr := require.True
	for val, check := range map[string]func(require.TestingT, bool, ...interface{}){
		// false checks
		CanonicalRootRole:      f,
		CanonicalTargetsRole:   f,
		CanonicalSnapshotRole:  f,
		CanonicalTimestampRole: f,
		"foo":   f,
		"foo/*": f,
		path.Join(CanonicalRootRole, "*"):           f,
		path.Join(CanonicalSnapshotRole, "*"):       f,
		path.Join(CanonicalTimestampRole, "*"):      f,
		path.Join(CanonicalTargetsRole, "*", "foo"): f,
		path.Join(CanonicalTargetsRole, "*", "*"):   f,
		fmt.Sprintf("%s//*", CanonicalTargetsRole):  f,
		fmt.Sprintf("%s/*//", CanonicalTargetsRole): f,
		fmt.Sprintf("%s/*/", CanonicalTargetsRole):  f,

		// true checks
		path.Join(CanonicalTargetsRole, "*"):        tr,
		path.Join(CanonicalTargetsRole, "foo", "*"): tr,
	} {
		check(t, IsWildDelegation(val))
	}
}

func TestValidRoleFunction(t *testing.T) {
	require.True(t, ValidRole(CanonicalRootRole))
	require.True(t, ValidRole(CanonicalTimestampRole))
	require.True(t, ValidRole(CanonicalSnapshotRole))
	require.True(t, ValidRole(CanonicalTargetsRole))
	require.True(t, ValidRole(path.Join(CanonicalTargetsRole, "level1")))
	require.True(t, ValidRole(
		path.Join(CanonicalTargetsRole, "level1", "level2", "level3")))

	require.False(t, ValidRole(""))
	require.False(t, ValidRole(CanonicalRootRole+"/"))
	require.False(t, ValidRole(CanonicalTimestampRole+"/"))
	require.False(t, ValidRole(CanonicalSnapshotRole+"/"))
	require.False(t, ValidRole(CanonicalTargetsRole+"/"))

	require.False(t, ValidRole(path.Join(CanonicalRootRole, "level1")))

	require.False(t, ValidRole(path.Join("role")))
}

func TestIsBaseRole(t *testing.T) {
	for _, role := range BaseRoles {
		require.True(t, IsBaseRole(role))
	}
	require.False(t, IsBaseRole("user"))
	require.False(t, IsBaseRole(
		path.Join(CanonicalTargetsRole, "level1", "level2", "level3")))
	require.False(t, IsBaseRole(path.Join(CanonicalTargetsRole, "level1")))
	require.False(t, IsBaseRole(""))
}

func TestBaseRoleEquals(t *testing.T) {
	fakeKeyHello := NewRSAPublicKey([]byte("hello"))
	fakeKeyThere := NewRSAPublicKey([]byte("there"))

	keys := map[string]PublicKey{"hello": fakeKeyHello, "there": fakeKeyThere}
	baseRole := BaseRole{Name: "name", Threshold: 1, Keys: keys}

	require.True(t, BaseRole{}.Equals(BaseRole{}))
	require.True(t, baseRole.Equals(BaseRole{Name: "name", Threshold: 1, Keys: keys}))
	require.False(t, baseRole.Equals(BaseRole{}))
	require.False(t, baseRole.Equals(BaseRole{Name: "notName", Threshold: 1, Keys: keys}))
	require.False(t, baseRole.Equals(BaseRole{Name: "name", Threshold: 2, Keys: keys}))
	require.False(t, baseRole.Equals(BaseRole{Name: "name", Threshold: 1,
		Keys: map[string]PublicKey{"hello": fakeKeyThere, "there": fakeKeyHello}}))
	require.False(t, baseRole.Equals(BaseRole{Name: "name", Threshold: 1,
		Keys: map[string]PublicKey{"hello": fakeKeyHello, "there": fakeKeyHello}}))
	require.False(t, baseRole.Equals(BaseRole{Name: "name", Threshold: 1,
		Keys: map[string]PublicKey{"hello": fakeKeyHello}}))
	require.False(t, baseRole.Equals(BaseRole{Name: "name", Threshold: 1,
		Keys: map[string]PublicKey{"hello": fakeKeyHello, "there": fakeKeyThere, "again": fakeKeyHello}}))
}
