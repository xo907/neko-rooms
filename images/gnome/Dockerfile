ARG BASE_IMAGE=ghcr.io/m1k1o/neko/base:3.1.6
FROM $BASE_IMAGE

# neko-rooms reads this to start the room with neko v3 settings
LABEL net.m1k1o.neko.api-version=3

#
# a minimal GNOME desktop as Debian ships it: the shell, files, a terminal and
# a browser, with Debian's theme and wallpaper. No display manager or systemd,
# the shell runs on neko's X server, see start-gnome.
RUN set -eux; apt-get update; \
    apt-get install -y --no-install-recommends \
        gnome-shell gnome-settings-daemon \
        nautilus gnome-terminal firefox-esr \
        desktop-base gnome-backgrounds adwaita-icon-theme fonts-cantarell \
        # software rendering, rooms have no GPU
        libgl1-mesa-dri \
        dbus at-spi2-core dconf-cli sudo; \
    #
    # X must keep running as the neko user, see the kde image
    apt-get remove -y xserver-xorg-legacy; \
    #
    # add user to sudoers
    usermod -aG sudo neko; \
    echo "neko:neko" | chpasswd; \
    echo "%sudo ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers; \
    # clean up
    apt-get clean -y; \
    rm -rf /var/lib/apt/lists/* /var/cache/apt/*

#
# defaults for a shared screen: no lock, no blanking, no welcome tour
COPY 90-neko-gnome.gschema.override /usr/share/glib-2.0/schemas/
RUN glib-compile-schemas /usr/share/glib-2.0/schemas/

#
# copy configuation files, and start the system bus the shell talks to
COPY --chmod=755 start-gnome /usr/local/bin/start-gnome
COPY supervisord.conf /etc/neko/supervisord/gnome.conf
RUN ln -s /etc/neko/supervisord.dbus.conf /etc/neko/supervisord/dbus.conf
