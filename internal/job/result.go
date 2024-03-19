package job

type resultAction byte

const resultNoAction resultAction = 0

const (
	resultSkipped resultAction = 1 << iota
	resultUpdated
	resultDeleted
	resultCreated
)

func (a resultAction) Updated() bool {
	return a&resultUpdated != 0
}

func (a resultAction) Skipped() bool {
	return a&resultSkipped != 0
}

func (a resultAction) Deleted() bool {
	return a&resultDeleted != 0
}

func (a resultAction) Created() bool {
	return a&resultCreated != 0
}

func (a resultAction) Changed() bool {
	return a > resultSkipped
}

func (a *resultAction) add(result resultAction) {
	*a = *a | result
}

func (a *resultAction) remove(result resultAction) {
	*a = *a &^ result
}
