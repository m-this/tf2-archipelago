package installer

// GitHub release assets are capped at 2 GiB each. Potato is split as raw byte
// ranges of the original ZIP, so joining these in order restores its exact
// bytes. docs/en/community-archives.md records the release and every digest.
const communityReleaseURL = "https://github.com/m-this/tf2-archipelago/releases/download/community-assets-2026-09-20/"

type communityPart struct {
	URL    string
	Size   int64
	SHA256 string
}

var communityGitHubParts = map[string][]communityPart{
	"archive-assets.zip": {
		{communityReleaseURL + "archive-assets.zip.part-000", 1_500_000_000, "de534578d4bba0ca7e0b5f44c3322981b6870339d363488a5f637883878d709f"},
		{communityReleaseURL + "archive-assets.zip.part-001", 1_094_253_886, "6de84051e4645aa358ab42391f0da8347f0a3eeca20cbf80dd81409f851c9cc6"},
	},
	"mlarchive-assets.zip": {
		{communityReleaseURL + "mlarchive-assets.zip", 190_820_351, "c6ba6c85c4466f012094388a59e15e4973c532cd1fd3bb2f404f1d7abc980149"},
	},
}
