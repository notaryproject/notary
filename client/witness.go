package client

func (r *NotaryRepository) Witness(role string) error {
	cl, err := changelist.NewFileChangelist(filepath.Join(r.tufRepoPath, "changelist"))
	if err != nil {
		return err
	}
	defer cl.Close()
	logrus.Debugf("Marking role %r for witnessing on next publish.\n", role)

	// scope is role
}
