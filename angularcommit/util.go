package angularcommit

// check if all change are in angular format
func checkAllAngularChanges(changes *[]*Change) bool {
	for _, c := range *changes {
		if !c.isAngular {
			return false
		}
	}
	return true
}

// Check if there is at least one type in changes
func hasChangeType(changes *[]*Change, t string) bool {
	for _, c := range *changes {
		if c.CommitType == t {
			return true
		}
	}
	return false
}
