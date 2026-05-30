package layout

func Calculate(width, height int, showSidebar bool) Dimensions {
	const (
		headerHeight    = 2
		footerHeight    = 2
		sidebarWidth    = 18
		sidebarMinWidth = 24
	)

	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}

	hdr := min(headerHeight, height)
	ftr := 0
	if height > hdr {
		ftr = min(footerHeight, height-hdr)
	}
	bodyHeight := max(0, height-hdr-ftr)

	sidebar := 0
	if showSidebar && width >= sidebarMinWidth {
		sidebar = min(sidebarWidth, width)
	}

	return Dimensions{
		Width:   width,
		Height:  height,
		Header:  Region{Width: width, Height: hdr},
		Sidebar: Region{Width: sidebar, Height: bodyHeight},
		Main:    Region{Width: max(0, width-sidebar), Height: bodyHeight},
		Footer:  Region{Width: width, Height: ftr},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
