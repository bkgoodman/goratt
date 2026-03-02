# TODO

One minor side note: processing.go has a background goroutine that draws (FillRect/FlushRect) without drawMu, which is a race condition on the gg.Context (not a deadlock). It won't cause a hang, but could theoretically cause a visual glitch. Not urgent.
