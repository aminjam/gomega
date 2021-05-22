package gmeasure

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

const CACHE_EXT = ".gmeasure-cache"

type ExperimentCache struct {
	Path string
}

func NewExperimentCache(path string) (ExperimentCache, error) {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		err := os.MkdirAll(path, 0777)
		if err != nil {
			return ExperimentCache{}, err
		}
	} else if !stat.IsDir() {
		return ExperimentCache{}, fmt.Errorf("%s is not a directory", path)
	}

	return ExperimentCache{
		Path: path,
	}, nil
}

type CachedExperimentHeader struct {
	Name    string
	Version int
}

func (cache ExperimentCache) hashOf(name string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(name)))
}

func (cache ExperimentCache) readHeader(filename string) (CachedExperimentHeader, error) {
	out := CachedExperimentHeader{}
	f, err := os.Open(filepath.Join(cache.Path, filename))
	if err != nil {
		return out, err
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(&out)
	return out, err
}

func (cache ExperimentCache) List() ([]CachedExperimentHeader, error) {
	out := []CachedExperimentHeader{}
	infos, err := ioutil.ReadDir(cache.Path)
	if err != nil {
		return out, err
	}
	for _, info := range infos {
		if filepath.Ext(info.Name()) != CACHE_EXT {
			continue
		}
		header, err := cache.readHeader(info.Name())
		if err != nil {
			return out, err
		}
		out = append(out, header)
	}
	return out, nil
}

func (cache ExperimentCache) Clear() error {
	infos, err := ioutil.ReadDir(cache.Path)
	if err != nil {
		return err
	}
	for _, info := range infos {
		if filepath.Ext(info.Name()) != CACHE_EXT {
			continue
		}
		err := os.Remove(filepath.Join(cache.Path, info.Name()))
		if err != nil {
			return err
		}
	}
	return nil
}

func (cache ExperimentCache) Load(name string, version int) *Experiment {
	path := filepath.Join(cache.Path, cache.hashOf(name)+CACHE_EXT)
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	header := CachedExperimentHeader{}
	dec.Decode(&header)
	if header.Version < version {
		return nil
	}
	out := NewExperiment("")
	err = dec.Decode(out)
	if err != nil {
		return nil
	}
	return out
}

func (cache ExperimentCache) Save(name string, version int, experiment *Experiment) error {
	path := filepath.Join(cache.Path, cache.hashOf(name)+CACHE_EXT)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	err = enc.Encode(CachedExperimentHeader{
		Name:    name,
		Version: version,
	})
	if err != nil {
		return err
	}
	return enc.Encode(experiment)
}

func (cache ExperimentCache) Delete(name string) error {
	path := filepath.Join(cache.Path, cache.hashOf(name)+CACHE_EXT)
	return os.Remove(path)
}
