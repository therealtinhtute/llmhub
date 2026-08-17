package updater

import "fmt"

// SelectAsset returns the asset whose name matches exactly, failing on any
// duplicate so ambiguous metadata can never be silently accepted.
func SelectAsset(rel Release, name string) (Asset, error) {
	var found *Asset
	for i := range rel.Assets {
		if rel.Assets[i].Name != name {
			continue
		}
		if found != nil {
			return Asset{}, fmt.Errorf("duplicate asset %q in release %s", name, rel.Tag)
		}
		found = &rel.Assets[i]
	}
	if found == nil {
		return Asset{}, fmt.Errorf("release %s has no asset %q", rel.Tag, name)
	}
	return *found, nil
}
