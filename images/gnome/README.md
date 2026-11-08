# GNOME room image

A minimal GNOME desktop for rooms, the way Debian 13 ships it: GNOME Shell 48,
Files, Terminal and Firefox ESR, with Debian's theme and wallpaper. Built on
`ghcr.io/m1k1o/neko/base` by [image_gnome.yml](../../.github/workflows/image_gnome.yml)
and published as `ghcr.io/xo907/neko-rooms/gnome:3.1.6` for amd64 and arm64.

- The shell runs on neko's X server without a display manager or systemd, see
  [start-gnome](start-gnome).
- Rooms have no GPU, so everything is drawn in software (llvmpipe). Give a
  GNOME room 2 CPUs or more; animations are off because they stutter when
  streamed.
- The screen never locks or blanks, and the theme is dark. Both are defaults
  in [90-neko-gnome.gschema.override](90-neko-gnome.gschema.override).
- The `neko` user can use `sudo`, like in neko's XFCE and KDE images.

To use it, pull it from **My rooms → Pull neko images** and pick it in
**New room**. The package must be public on GitHub (Packages → gnome →
Package settings → Change visibility), or the server must be logged in to
`ghcr.io`.
