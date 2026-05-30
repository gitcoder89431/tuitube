package theme

func BuiltIns() []Theme {
	return []Theme{
		Phosphor(),
		Dracula(),
		TokyoNight(),
		TokyoNightStorm(),
		CatppuccinMocha(),
		CatppuccinMacchiato(),
		Nord(),
		GruvboxDark(),
		GruvboxLight(),
		Kanagawa(),
		RosePine(),
		RosePineMoon(),
		SolarizedDark(),
		SolarizedLight(),
		OneDark(),
		EverforestDark(),
		Monokai(),
	}
}

func Next(current string) Theme {
	themes := BuiltIns()
	if len(themes) == 0 {
		return Theme{}
	}
	for i, t := range themes {
		if t.Name == current {
			return themes[(i+1)%len(themes)]
		}
	}
	return themes[0]
}
