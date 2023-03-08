package transform

/*
func (t Transformer) ProfilterToSqlite() (err error) {

	// get all profiles
	pfps, err := t.Neo.GetProfileFilterPlatformPairs()
	if err != nil {
		logrus.Fatal(err)
	}
	for idx, pfp := range pfps {
		p := pfp.Profile
		profileID, found, err := t.SQlite.Profile.HasProfile(p.Name)
		if err != nil {
			err = errors.Wrap(err, "t.SQlite.Profile.HasProfile(p.Name)")
			logrus.Error(err)
			continue
		}
		if !found {
			err = t.SQlite.Profile.Insert(&p)
			if err != nil {
				err = errors.Wrap(err, "t.SQlite.Profile.Insert(&p")
				logrus.Fatal(err)
			}
		} else {
			p.ID = profileID
		}
		pfp.Profile = p
		f := pfp.Filter
		err = t.SQlite.Filter.InsertFilter(&f)
		if err != nil {
			err = errors.Wrap(err, "t.SQlite.Filter.InsertFilter(&f)")
			logrus.Fatal(err)
		}
		pfp.Filter = f
		// Insert platform
		plat := pfp.Platform
		err = t.SQlite.Platform.Upsert(&plat)
		if err != nil {
			err = errors.Wrap(err, "t.SQlite.Platform.Upsert(&plat)")
			logrus.Fatal(err)
		}
		pfp.Platform = plat
		// insert profile_filter
		err = t.SQlite.Filter.InsertProfileFilter(pfp)
		if err != nil {
			err = errors.Wrap(err, "t.SQlite.Filter.InsertProfileFilter(pfp)")
			logrus.Fatal(err)
		}
		// insert platform_filter
		err = t.SQlite.Platform.InsertPlatformFilter(pfp)
		if err != nil {
			err = errors.Wrap(err, "t.SQlite.Platform.InsertPlatformFilter(pfp)")
			logrus.Fatal(err)
		}
		pfps[idx] = pfp
	}
	return
}
*/
