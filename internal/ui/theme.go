package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type workbenchTheme struct {
	base fyne.Theme
}

func newWorkbenchTheme() fyne.Theme {
	return workbenchTheme{base: theme.DarkTheme()}
}

func (t workbenchTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0x11, G: 0x15, B: 0x1B, A: 0xFF}
	case theme.ColorNameHeaderBackground, theme.ColorNameInputBackground:
		return color.NRGBA{R: 0x17, G: 0x1C, B: 0x23, A: 0xFF}
	case theme.ColorNameButton, theme.ColorNameMenuBackground:
		return color.NRGBA{R: 0x1D, G: 0x24, B: 0x2D, A: 0xFF}
	case theme.ColorNameInnerWindowBorder, theme.ColorNameInnerWindowBorderInactive, theme.ColorNameInputBorder, theme.ColorNameSeparator:
		return color.NRGBA{R: 0x2A, G: 0x32, B: 0x3D, A: 0xFF}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0xF1, G: 0xF4, B: 0xF7, A: 0xFF}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 0x6E, G: 0x77, B: 0x82, A: 0xFF}
	case theme.ColorNameHover:
		return color.NRGBA{R: 0x2A, G: 0x32, B: 0x3D, A: 0xFF}
	case theme.ColorNameSelection:
		return color.NRGBA{R: 0x22, G: 0x3A, B: 0x5A, A: 0xFF}
	case theme.ColorNameFocus, theme.ColorNamePrimary, theme.ColorNameHyperlink:
		return color.NRGBA{R: 0x3B, G: 0x82, B: 0xF6, A: 0xFF}
	case theme.ColorNameForegroundOnPrimary:
		return color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 0x58, G: 0x64, B: 0x74, A: 0xFF}
	case theme.ColorNameScrollBarBackground:
		return color.NRGBA{R: 0x17, G: 0x1C, B: 0x23, A: 0xFF}
	}
	return t.base.Color(name, variant)
}

func (t workbenchTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.base.Font(style)
}

func (t workbenchTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}

func (t workbenchTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameCaptionText:
		return 12
	case theme.SizeNameText:
		return 14
	case theme.SizeNameSubHeadingText:
		return 16
	case theme.SizeNameHeadingText:
		return 20
	case theme.SizeNameInnerPadding:
		return 6
	case theme.SizeNamePadding:
		return 6
	case theme.SizeNameLineSpacing:
		return 2
	case theme.SizeNameSeparatorThickness, theme.SizeNameSplitThickness:
		return 1
	case theme.SizeNameButtonRadius, theme.SizeNameCardRadius, theme.SizeNameSelectionRadius:
		return 5
	case theme.SizeNameScrollBar:
		return 10
	case theme.SizeNameScrollBarSmall:
		return 6
	case theme.SizeNameScrollBarRadius:
		return 4
	}
	return t.base.Size(name)
}
