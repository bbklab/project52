package main

import (
	"fmt"
	"net/url"

	"github.com/andygrunwald/megos"
)

func MesosState() (*megos.State, error) {
	var (
		u, err = url.Parse("http://" + addr)
		client = megos.NewClient([]*url.URL{u}, nil)
	)
	if err != nil {
		return nil, err
	}

	return client.GetStateFromCluster()
}

func FrameworkState() (*megos.Framework, error) {
	stats, err := MesosState()
	if err != nil {
		return nil, err
	}

	for _, fw := range stats.Frameworks {
		if fw.Name == "bbklab" {
			return &fw, nil
		}
	}

	return nil, fmt.Errorf("no such framework: bbklab")
}
