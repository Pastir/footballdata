package types

type (
	ErrorApi struct {
		Err struct {
			Message *string `json:"message"`
			Code    *int    `json:"code"`
		} `json:"error"`
	}
	//Pagination структура содержит мета данные о пагинации
	Pagination struct {
		Total       *int `json:"total"`
		Count       *int `json:"count"`
		PerPage     *int `json:"per_page"`
		CurrentPage *int `json:"current_page"`
		TotalPages  *int `json:"total_pages"`
		Links       struct {
			Next *string `json:"next"`
		} `json:"links"`
	}
	//метаданные
	meta struct {
		Pagination Pagination `json:"pagination"`
	}
)
