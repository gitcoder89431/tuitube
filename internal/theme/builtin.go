package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type palette struct {
	name       string
	bg         color.Color
	fg         color.Color
	muted      color.Color
	subtle     color.Color
	selected   color.Color
	divider    color.Color
	accent     color.Color
	accentAlt  color.Color
	chrome     color.Color
	info       color.Color
	success    color.Color
	warn       color.Color
	errorColor color.Color
}

func newTheme(p palette) Theme {
	chrome := p.accent
	if p.chrome != nil {
		chrome = p.chrome
	}

	return Theme{
		Name:          p.name,
		Background:    p.bg,
		Text:          lipgloss.NewStyle().Foreground(p.fg).Background(p.bg),
		Muted:         lipgloss.NewStyle().Foreground(p.muted).Background(p.bg),
		Accent:        lipgloss.NewStyle().Foreground(p.accent).Background(p.bg).Bold(true),
		Title:         lipgloss.NewStyle().Bold(true).Foreground(chrome).Background(p.bg),
		Selected:      lipgloss.NewStyle().Bold(true).Foreground(p.fg).Background(p.selected),
		Disabled:      lipgloss.NewStyle().Foreground(p.subtle).Background(p.bg),
		Header:        lipgloss.NewStyle().Foreground(p.fg).Background(p.bg).Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(p.divider).Padding(0, 1),
		HeaderAccent:  lipgloss.NewStyle().Foreground(chrome).Background(p.bg).Bold(true),
		Sidebar:       lipgloss.NewStyle().Background(p.bg).Border(lipgloss.NormalBorder(), false, true, false, false).BorderForeground(p.divider).Padding(1, 1),
		Main:          lipgloss.NewStyle().Foreground(p.fg).Background(p.bg).Padding(1, 2),
		Footer:        lipgloss.NewStyle().Foreground(p.subtle).Background(p.bg).Border(lipgloss.NormalBorder(), true, false, false, false).BorderForeground(p.divider).Padding(0, 1),
		Modal:         lipgloss.NewStyle().Foreground(p.fg).Background(p.bg).Border(lipgloss.RoundedBorder()).BorderForeground(p.accentAlt).Padding(1, 2),
		PaletteAccent: lipgloss.NewStyle().Foreground(p.accentAlt).Background(p.bg).Bold(true),
		Border:        lipgloss.NewStyle().Foreground(p.divider).Background(p.bg),
		Focus:         lipgloss.NewStyle().Foreground(p.accent).Background(p.bg),
		Info:          lipgloss.NewStyle().Foreground(p.info).Background(p.bg),
		Success:       lipgloss.NewStyle().Foreground(p.success).Background(p.bg),
		Warn:          lipgloss.NewStyle().Foreground(p.warn).Background(p.bg),
		Error:         lipgloss.NewStyle().Foreground(p.errorColor).Background(p.bg),
	}
}

func Phosphor() Theme {
	return newTheme(palette{
		name:       "Phosphor",
		bg:         lipgloss.Color("#111A2C"),
		fg:         lipgloss.Color("#9FE8B0"),
		muted:      lipgloss.Color("#5A8A68"),
		subtle:     lipgloss.Color("#788E80"),
		selected:   lipgloss.Color("#18233D"),
		divider:    lipgloss.Color("#2A3752"),
		accent:     lipgloss.Color("#6FD0E3"),
		accentAlt:  lipgloss.Color("#FFB347"),
		chrome:     lipgloss.Color("#FFB347"),
		info:       lipgloss.Color("#6FD0E3"),
		success:    lipgloss.Color("#9FE8B0"),
		warn:       lipgloss.Color("#FFB347"),
		errorColor: lipgloss.Color("#FF6B7A"),
	})
}

func Dracula() Theme {
	return newTheme(palette{
		name:       "Dracula",
		bg:         lipgloss.Color("#282A36"),
		fg:         lipgloss.Color("#F8F8F2"),
		muted:      lipgloss.Color("#6272A4"),
		subtle:     lipgloss.Color("#44475A"),
		selected:   lipgloss.Color("#44475A"),
		divider:    lipgloss.Color("#6272A4"),
		accent:     lipgloss.Color("#BD93F9"),
		accentAlt:  lipgloss.Color("#FF79C6"),
		info:       lipgloss.Color("#8BE9FD"),
		success:    lipgloss.Color("#50FA7B"),
		warn:       lipgloss.Color("#F1FA8C"),
		errorColor: lipgloss.Color("#FF5555"),
	})
}

func TokyoNight() Theme {
	return newTheme(palette{
		name:       "Tokyo Night",
		bg:         lipgloss.Color("#1A1B26"),
		fg:         lipgloss.Color("#C0CAF5"),
		muted:      lipgloss.Color("#565F89"),
		subtle:     lipgloss.Color("#414868"),
		selected:   lipgloss.Color("#283457"),
		divider:    lipgloss.Color("#3B4261"),
		accent:     lipgloss.Color("#7AA2F7"),
		accentAlt:  lipgloss.Color("#BB9AF7"),
		info:       lipgloss.Color("#7DCFFF"),
		success:    lipgloss.Color("#9ECE6A"),
		warn:       lipgloss.Color("#E0AF68"),
		errorColor: lipgloss.Color("#F7768E"),
	})
}

func TokyoNightStorm() Theme {
	return newTheme(palette{
		name:       "Tokyo Night Storm",
		bg:         lipgloss.Color("#24283B"),
		fg:         lipgloss.Color("#C0CAF5"),
		muted:      lipgloss.Color("#565F89"),
		subtle:     lipgloss.Color("#414868"),
		selected:   lipgloss.Color("#2E3C64"),
		divider:    lipgloss.Color("#3B4261"),
		accent:     lipgloss.Color("#7AA2F7"),
		accentAlt:  lipgloss.Color("#BB9AF7"),
		info:       lipgloss.Color("#7DCFFF"),
		success:    lipgloss.Color("#9ECE6A"),
		warn:       lipgloss.Color("#E0AF68"),
		errorColor: lipgloss.Color("#F7768E"),
	})
}

func CatppuccinMocha() Theme {
	return newTheme(palette{
		name:       "Catppuccin Mocha",
		bg:         lipgloss.Color("#1E1E2E"),
		fg:         lipgloss.Color("#CDD6F4"),
		muted:      lipgloss.Color("#7F849C"),
		subtle:     lipgloss.Color("#6C7086"),
		selected:   lipgloss.Color("#313244"),
		divider:    lipgloss.Color("#45475A"),
		accent:     lipgloss.Color("#89B4FA"),
		accentAlt:  lipgloss.Color("#CBA6F7"),
		info:       lipgloss.Color("#89DCEB"),
		success:    lipgloss.Color("#A6E3A1"),
		warn:       lipgloss.Color("#F9E2AF"),
		errorColor: lipgloss.Color("#F38BA8"),
	})
}

func CatppuccinMacchiato() Theme {
	return newTheme(palette{
		name:       "Catppuccin Macchiato",
		bg:         lipgloss.Color("#24273A"),
		fg:         lipgloss.Color("#CAD3F5"),
		muted:      lipgloss.Color("#8087A2"),
		subtle:     lipgloss.Color("#6E738D"),
		selected:   lipgloss.Color("#363A4F"),
		divider:    lipgloss.Color("#494D64"),
		accent:     lipgloss.Color("#8AADF4"),
		accentAlt:  lipgloss.Color("#C6A0F6"),
		info:       lipgloss.Color("#91D7E3"),
		success:    lipgloss.Color("#A6DA95"),
		warn:       lipgloss.Color("#EED49F"),
		errorColor: lipgloss.Color("#ED8796"),
	})
}

func Nord() Theme {
	return newTheme(palette{
		name:       "Nord",
		bg:         lipgloss.Color("#2E3440"),
		fg:         lipgloss.Color("#D8DEE9"),
		muted:      lipgloss.Color("#81A1C1"),
		subtle:     lipgloss.Color("#4C566A"),
		selected:   lipgloss.Color("#3B4252"),
		divider:    lipgloss.Color("#4C566A"),
		accent:     lipgloss.Color("#88C0D0"),
		accentAlt:  lipgloss.Color("#B48EAD"),
		info:       lipgloss.Color("#81A1C1"),
		success:    lipgloss.Color("#A3BE8C"),
		warn:       lipgloss.Color("#EBCB8B"),
		errorColor: lipgloss.Color("#BF616A"),
	})
}

func GruvboxDark() Theme {
	return newTheme(palette{
		name:       "Gruvbox Dark",
		bg:         lipgloss.Color("#282828"),
		fg:         lipgloss.Color("#EBDBB2"),
		muted:      lipgloss.Color("#928374"),
		subtle:     lipgloss.Color("#665C54"),
		selected:   lipgloss.Color("#3C3836"),
		divider:    lipgloss.Color("#504945"),
		accent:     lipgloss.Color("#83A598"),
		accentAlt:  lipgloss.Color("#D3869B"),
		info:       lipgloss.Color("#8EC07C"),
		success:    lipgloss.Color("#B8BB26"),
		warn:       lipgloss.Color("#FABD2F"),
		errorColor: lipgloss.Color("#FB4934"),
	})
}

func GruvboxLight() Theme {
	return newTheme(palette{
		name:       "Gruvbox Light",
		bg:         lipgloss.Color("#FBF1C7"),
		fg:         lipgloss.Color("#3C3836"),
		muted:      lipgloss.Color("#7C6F64"),
		subtle:     lipgloss.Color("#928374"),
		selected:   lipgloss.Color("#EBDBB2"),
		divider:    lipgloss.Color("#D5C4A1"),
		accent:     lipgloss.Color("#076678"),
		accentAlt:  lipgloss.Color("#8F3F71"),
		info:       lipgloss.Color("#427B58"),
		success:    lipgloss.Color("#79740E"),
		warn:       lipgloss.Color("#B57614"),
		errorColor: lipgloss.Color("#9D0006"),
	})
}

func Kanagawa() Theme {
	return newTheme(palette{
		name:       "Kanagawa",
		bg:         lipgloss.Color("#1F1F28"),
		fg:         lipgloss.Color("#DCD7BA"),
		muted:      lipgloss.Color("#727169"),
		subtle:     lipgloss.Color("#54546D"),
		selected:   lipgloss.Color("#2A2A37"),
		divider:    lipgloss.Color("#54546D"),
		accent:     lipgloss.Color("#7E9CD8"),
		accentAlt:  lipgloss.Color("#957FB8"),
		info:       lipgloss.Color("#7FB4CA"),
		success:    lipgloss.Color("#98BB6C"),
		warn:       lipgloss.Color("#E6C384"),
		errorColor: lipgloss.Color("#E82424"),
	})
}

func RosePine() Theme {
	return newTheme(palette{
		name:       "Rose Pine",
		bg:         lipgloss.Color("#191724"),
		fg:         lipgloss.Color("#E0DEF4"),
		muted:      lipgloss.Color("#6E6A86"),
		subtle:     lipgloss.Color("#908CAA"),
		selected:   lipgloss.Color("#26233A"),
		divider:    lipgloss.Color("#403D52"),
		accent:     lipgloss.Color("#9CCFD8"),
		accentAlt:  lipgloss.Color("#C4A7E7"),
		info:       lipgloss.Color("#31748F"),
		success:    lipgloss.Color("#9CCFD8"),
		warn:       lipgloss.Color("#F6C177"),
		errorColor: lipgloss.Color("#EB6F92"),
	})
}

func RosePineMoon() Theme {
	return newTheme(palette{
		name:       "Rose Pine Moon",
		bg:         lipgloss.Color("#232136"),
		fg:         lipgloss.Color("#E0DEF4"),
		muted:      lipgloss.Color("#6E6A86"),
		subtle:     lipgloss.Color("#908CAA"),
		selected:   lipgloss.Color("#2A273F"),
		divider:    lipgloss.Color("#44415A"),
		accent:     lipgloss.Color("#9CCFD8"),
		accentAlt:  lipgloss.Color("#C4A7E7"),
		info:       lipgloss.Color("#3E8FB0"),
		success:    lipgloss.Color("#9CCFD8"),
		warn:       lipgloss.Color("#F6C177"),
		errorColor: lipgloss.Color("#EB6F92"),
	})
}

func SolarizedDark() Theme {
	return newTheme(palette{
		name:       "Solarized Dark",
		bg:         lipgloss.Color("#002B36"),
		fg:         lipgloss.Color("#839496"),
		muted:      lipgloss.Color("#586E75"),
		subtle:     lipgloss.Color("#657B83"),
		selected:   lipgloss.Color("#073642"),
		divider:    lipgloss.Color("#586E75"),
		accent:     lipgloss.Color("#268BD2"),
		accentAlt:  lipgloss.Color("#6C71C4"),
		info:       lipgloss.Color("#2AA198"),
		success:    lipgloss.Color("#859900"),
		warn:       lipgloss.Color("#B58900"),
		errorColor: lipgloss.Color("#DC322F"),
	})
}

func SolarizedLight() Theme {
	return newTheme(palette{
		name:       "Solarized Light",
		bg:         lipgloss.Color("#FDF6E3"),
		fg:         lipgloss.Color("#657B83"),
		muted:      lipgloss.Color("#93A1A1"),
		subtle:     lipgloss.Color("#839496"),
		selected:   lipgloss.Color("#EEE8D5"),
		divider:    lipgloss.Color("#93A1A1"),
		accent:     lipgloss.Color("#268BD2"),
		accentAlt:  lipgloss.Color("#6C71C4"),
		info:       lipgloss.Color("#2AA198"),
		success:    lipgloss.Color("#859900"),
		warn:       lipgloss.Color("#B58900"),
		errorColor: lipgloss.Color("#DC322F"),
	})
}

func OneDark() Theme {
	return newTheme(palette{
		name:       "One Dark",
		bg:         lipgloss.Color("#282C34"),
		fg:         lipgloss.Color("#ABB2BF"),
		muted:      lipgloss.Color("#5C6370"),
		subtle:     lipgloss.Color("#4B5263"),
		selected:   lipgloss.Color("#3E4451"),
		divider:    lipgloss.Color("#4B5263"),
		accent:     lipgloss.Color("#61AFEF"),
		accentAlt:  lipgloss.Color("#C678DD"),
		info:       lipgloss.Color("#56B6C2"),
		success:    lipgloss.Color("#98C379"),
		warn:       lipgloss.Color("#E5C07B"),
		errorColor: lipgloss.Color("#E06C75"),
	})
}

func EverforestDark() Theme {
	return newTheme(palette{
		name:       "Everforest Dark",
		bg:         lipgloss.Color("#2D353B"),
		fg:         lipgloss.Color("#D3C6AA"),
		muted:      lipgloss.Color("#859289"),
		subtle:     lipgloss.Color("#7A8478"),
		selected:   lipgloss.Color("#343F44"),
		divider:    lipgloss.Color("#475258"),
		accent:     lipgloss.Color("#7FBBB3"),
		accentAlt:  lipgloss.Color("#D699B6"),
		info:       lipgloss.Color("#83C092"),
		success:    lipgloss.Color("#A7C080"),
		warn:       lipgloss.Color("#DBBC7F"),
		errorColor: lipgloss.Color("#E67E80"),
	})
}

func Monokai() Theme {
	return newTheme(palette{
		name:       "Monokai",
		bg:         lipgloss.Color("#272822"),
		fg:         lipgloss.Color("#F8F8F2"),
		muted:      lipgloss.Color("#75715E"),
		subtle:     lipgloss.Color("#49483E"),
		selected:   lipgloss.Color("#3E3D32"),
		divider:    lipgloss.Color("#49483E"),
		accent:     lipgloss.Color("#66D9EF"),
		accentAlt:  lipgloss.Color("#AE81FF"),
		info:       lipgloss.Color("#66D9EF"),
		success:    lipgloss.Color("#A6E22E"),
		warn:       lipgloss.Color("#E6DB74"),
		errorColor: lipgloss.Color("#F92672"),
	})
}
