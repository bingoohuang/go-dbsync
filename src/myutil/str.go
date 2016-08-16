package myutil

func RowToString(cols []string, m *map[string]string) string {
	mystr := MyStr{}
	mystr.PS("{")
	for _, col := range cols {
		val, ok := (*m)[col]
		if ok {
			mystr.PKV(col, val)
			delete(*m, col)
		}
	}

	return mystr.PM(*m).PS("}").Str()
}
