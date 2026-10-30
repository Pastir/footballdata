package types

type (
	extra struct {
		SubRegion   *string `json:"sub_region"`
		WorldRegion *string `json:"world_region"`
		Fifa        *string `json:"fifa"`
		Iso         *string `json:"iso"`
		Iso2        *string `json:"iso2"`
		Longitude   *string `json:"longitude"`
		Latitude    *string `json:"latitude"`
		Flag        *string `json:"flag"`
	}

	Country struct {
		Id        *int    `json:"id"`
		Name      *string `json:"name"`
		ImagePath *string `json:"image_path"`
		Extra     extra   `json:"extra"`
	}

	CountryData struct {
		Countries map[string][]CountryName
	}

	CountryName struct {
		ProviderId        int
		ProviderCountryId int
		CountryISO        string
		CountryCode       string
		CountryName       string
		Uid               string
	}
)
