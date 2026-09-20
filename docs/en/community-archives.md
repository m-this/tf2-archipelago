# Community archive snapshots

The launcher expects these **complete ZIP files, including maps and NAV files**. SHA-256 covers the ZIP bytes, not the extracted files. These are the snapshots used to review the mission catalog; changing an archive can change a mission without changing its name.

These archives contain community made maps, missions, and supporting assets from Potato and Moonlight. Credit belongs to their creators; the [Potato Archive creators page](https://potato.tf/creators) lists mission authors by campaign. The original archive project is [potato-tf/ArchiveAssets](https://github.com/potato-tf/ArchiveAssets). These links identify the source and authors; they do not establish redistribution permission.

The launcher downloads these snapshots from the [community assets GitHub release](https://github.com/m-this/tf2-archipelago/releases/tag/community-assets-2026-09-20) first. The original Potato endpoints below are fallback mirrors. The Potato ZIP exceeds GitHub's per-file limit, so its release assets are raw byte chunks. Concatenate them in numbered order to restore the ZIP; do not unzip either part separately.

The release tag is pinned deliberately: a new snapshot needs a new release, reviewed full and part hashes, and a launcher update. If the pinned release is unavailable or its bytes fail verification, the launcher logs the reason and tries Potato. It still checks the final ZIP against the pinned hash before installation.

| Pack | Source | Bytes | SHA-256 |
| --- | --- | ---: | --- |
| Potato `archive-assets.zip` | `https://dlarchive.potato.tf/archive-assets.zip` | 2,594,253,886 | `e7e54f3167b97341d11cf1a1b30f437bf0651fec40e4e1d25232b883cf44bb69` |
| Moonlight `mlarchive-assets.zip` | `https://dlml.potato.tf/mlarchive-assets.zip` | 190,820,351 | `c6ba6c85c4466f012094388a59e15e4973c532cd1fd3bb2f404f1d7abc980149` |

| GitHub release asset | Bytes | SHA-256 |
| --- | ---: | --- |
| `archive-assets.zip.part-000` | 1,500,000,000 | `de534578d4bba0ca7e0b5f44c3322981b6870339d363488a5f637883878d709f` |
| `archive-assets.zip.part-001` | 1,094,253,886 | `6de84051e4645aa358ab42391f0da8347f0a3eeca20cbf80dd81409f851c9cc6` |
| `mlarchive-assets.zip` | 190,820,351 | `c6ba6c85c4466f012094388a59e15e4973c532cd1fd3bb2f404f1d7abc980149` |

On Linux, `cat archive-assets.zip.part-000 archive-assets.zip.part-001 > archive-assets.zip` reconstructs Potato. On Windows, run `copy /b archive-assets.zip.part-000+archive-assets.zip.part-001 archive-assets.zip` in Command Prompt. The launcher assembles and verifies it automatically.

To check a downloaded file yourself, run `sha256sum archive-assets.zip mlarchive-assets.zip` on Linux or `Get-FileHash archive-assets.zip -Algorithm SHA256` in PowerShell. Compare with the full hashes above. The launcher checks these hashes when it downloads or imports a pack and again before installation. A differing ZIP is held as `*.hash-mismatch`; the Missions page shows an **Ignore hash mismatch** confirmation if you decide to use those exact bytes despite possible missing missions or server instability. Any later file change requires a new confirmation.
