package sanitize

import "strings"

// Description очищает описание от потенциально опасных HTML тегов
func Description(input string) string {
	// Заменяем < и > на безопасные HTML-сущности
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	return input
}
