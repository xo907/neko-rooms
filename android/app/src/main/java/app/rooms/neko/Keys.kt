package app.rooms.neko

// X11 keysyms the remote browser understands.
object Keys {
    const val BACKSPACE = 0xff08
    const val TAB = 0xff09
    const val RETURN = 0xff0d
    const val ESCAPE = 0xff1b
    const val DELETE = 0xffff
    const val HOME = 0xff50
    const val LEFT = 0xff51
    const val UP = 0xff52
    const val RIGHT = 0xff53
    const val DOWN = 0xff54
    const val PAGE_UP = 0xff55
    const val PAGE_DOWN = 0xff56
    const val END = 0xff57
    const val SHIFT = 0xffe1
    const val CONTROL = 0xffe3
    const val ALT = 0xffe9

    // keysym for a typed character: Latin-1 maps directly, the rest is
    // Unicode with the 0x01000000 offset
    fun forChar(c: Int): Int = when {
        c == '\n'.code -> RETURN
        c == '\t'.code -> TAB
        c in 0x20..0x7e || c in 0xa0..0xff -> c
        else -> 0x01000000 + c
    }
}

// mouse buttons
object Buttons {
    const val LEFT = 1
    const val MIDDLE = 2
    const val RIGHT = 3
}
