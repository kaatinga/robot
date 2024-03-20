package job

type resultAction byte

const resultNoAction resultAction = 0

const (
	resultSkipped resultAction = 1 << iota
	resultUpdated
	resultDeleted
	resultCreated
)

func (a resultAction) PrintAll() []string {
	var output []string
	for i := resultUpdated; i <= resultCreated; i <<= 1 {
		if result := a & i; result != 0 {
			output = append(output, result.String())
		}
	}
	return output
}

func (a resultAction) String() string {
	switch a {
	case resultSkipped:
		return "Skipped"
	case resultUpdated:
		return "Updated"
	case resultDeleted:
		return "Deleted"
	case resultCreated:
		return "Created"
	case resultNoAction:
		return "No action"
	default:
		return "Mixed action"
	}
}

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
