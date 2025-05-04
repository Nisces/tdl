package dl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/go-faster/errors"
	jsoniter "github.com/json-iterator/go"

	"github.com/iyear/tdl/core/downloader"
	"github.com/iyear/tdl/core/util/fsutil"
)

type progress struct {
	opts Options

	it *iter
}

func newProgress(it *iter, opts Options) *progress {
	return &progress{
		opts: opts,
		it:   it,
	}
}

func (p *progress) OnAdd(elem downloader.Elem) {
	e := elem.(*iterElem)
	data := map[string]any{
		"id":         e.id,
		"url":        e.url,
		"message_id": e.fromMsg.ID,
		"state":      "start",
		"total":      e.file.Size,
		"downloaded": 0,
	}
	fmt.Println(toJson(data))
}

func (p *progress) OnDownload(elem downloader.Elem, state downloader.ProgressState) {
	e := elem.(*iterElem)
	data := map[string]any{
		"id":         e.id,
		"url":        e.url,
		"message_id": e.fromMsg.ID,
		"state":      "downloading",
		"total":      state.Total,
		"downloaded": state.Downloaded,
	}
	fmt.Println(toJson(data))
}

func (p *progress) OnDone(elem downloader.Elem, err error) {
	e := elem.(*iterElem)
	if err := e.to.Close(); err != nil {
		p.fail(elem, errors.Wrap(err, "close file"))
		return
	}

	if err != nil {
		if !errors.Is(err, context.Canceled) { // don't report user cancel
			p.fail(elem, errors.Wrap(err, "progress"))
		}
		_ = os.Remove(e.to.Name()) // just try to remove temp file, ignore error
		return
	}

	p.it.Finish(e.id)

	if err := p.donePost(e); err != nil {
		p.fail(elem, errors.Wrap(err, "post file"))
		return
	}

	// 最后才成功，但上面这些东西本不该出现在这里
	data := map[string]any{
		"id":         e.id,
		"url":        e.url,
		"message_id": e.fromMsg.ID,
		"state":      "done",
		"total":      e.file.Size,
		"downloaded": e.file.Size,
	}
	fmt.Println(toJson(data))
}

func (p *progress) donePost(elem *iterElem) error {
	newfile := strings.TrimSuffix(filepath.Base(elem.to.Name()), tempExt)

	if p.opts.RewriteExt {
		mime, err := mimetype.DetectFile(elem.to.Name())
		if err != nil {
			return errors.Wrap(err, "detect mime")
		}
		ext := mime.Extension()
		if ext != "" && (filepath.Ext(newfile) != ext) {
			newfile = fsutil.GetNameWithoutExt(newfile) + ext
		}
	}

	if err := os.Rename(elem.to.Name(), filepath.Join(filepath.Dir(elem.to.Name()), newfile)); err != nil {
		return errors.Wrap(err, "rename file")
	}

	return nil
}

func (p *progress) fail(elem downloader.Elem, err error) {
	e := elem.(*iterElem)
	data := map[string]any{
		"id":         e.id,
		"url":        e.url,
		"message_id": e.fromMsg.ID,
		"state":      "fail",
		"total":      e.file.Size,
	}
	if err != nil {
		data["err_msg"] = err.Error()
	}
	fmt.Println(toJson(data))
}

func toJson(v any) string {
	j, _ := jsoniter.ConfigCompatibleWithStandardLibrary.MarshalToString(v)
	return j
}
