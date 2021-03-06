// Copyright 2016 Weaveworks Ltd.
// Use of this source code is governed by a Apache License Version 2.0 license
// that can be found at https://github.com/weaveworks/flux/blob/master/LICENSE

// 注意，这里面的这些方法看不懂作用是什么的，根据方法里面的参数，在某个git管理的项目下执行就能知道效果是什么了
package git

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"runtime"
	"strings"
)

// If true, every git invocation will be echoed to stdout
const trace = false

func config(ctx context.Context, workingDir, user, email string) error {
	for k, v := range map[string]string{
		"user.name":  user,
		"user.email": email,
	} {
		if err := execGitCmd(ctx, workingDir, nil, "config", k, v); err != nil {
			return errors.Wrap(err, "setting git config")
		}
	}
	return nil
}

func clone(ctx context.Context, workingDir, repoURL, repoBranch string) (path string, err error) {
	repoPath := workingDir
	args := []string{"clone"}
	if repoBranch != "" {
		args = append(args, "--branch", repoBranch)
	}
	args = append(args, repoURL, repoPath)
	if err := execGitCmd(ctx, workingDir, nil, args...); err != nil {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return "", errors.Wrap(err, "git clone")
	}
	return repoPath, nil
}

func mirror(ctx context.Context, workingDir, repoURL string) (path string, err error) {
	repoPath := workingDir
	args := []string{"clone", "--mirror"}
	args = append(args, repoURL, repoPath)
	if err := execGitCmd(ctx, workingDir, nil, args...); err != nil {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return "", errors.Wrap(err, "git clone --mirror")
	}
	return repoPath, nil
}

// checkPush sanity-checks that we can write to the upstream repo
// (being able to `clone` is an adequate check that we can read the
// upstream).
func checkPush(ctx context.Context, workingDir, upstream string) error {
	// --force just in case we fetched the tag from upstream when cloning
	if err := execGitCmd(ctx, workingDir, nil, "tag", "--force", CheckPushTag); err != nil {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return errors.Wrap(err, "tag for write check")
	}
	if err := execGitCmd(ctx, workingDir, nil, "push", "--force", upstream, "tag", CheckPushTag); err != nil {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return errors.Wrap(err, "attempt to push tag")
	}
	if err := execGitCmd(ctx, workingDir, nil, "push", "--delete", upstream, "tag", CheckPushTag); err != nil {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return err
	}
	return nil
}

func commit(ctx context.Context, workingDir string, commitAction CommitAction) error {
	commitAuthor := commitAction.Author
	if commitAuthor != "" {
		if err := execGitCmd(ctx,
			workingDir, nil,
			"commit",
			"--no-verify", "-a", "--author", commitAuthor, "-m", commitAction.Message,
		); err != nil {
			buf := make([]byte, 1024)
			outPut := make([]byte, 1024+len(err.Error()))
			strReader := strings.NewReader(err.Error() + "\n")
			strReader.Read(outPut)
			for {
				n := runtime.Stack(buf, false)
				if n < len(buf) {
					outPut = append(outPut, buf...)
					glog.Info(string(outPut))
					break
				}
				buf = make([]byte, 2*len(buf))
			}
			return errors.Wrap(err, "git commit")
		}
		return nil
	}
	if err := execGitCmd(ctx,
		workingDir, nil,
		"commit",
		"--no-verify", "-a", "-m", commitAction.Message,
	); err != nil {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return errors.Wrap(err, "git commit")
	}
	return nil
}

// push the refs given to the upstream repo
func push(ctx context.Context, workingDir, upstream string, refs []string) error {
	args := append([]string{"push", upstream}, refs...)
	if err := execGitCmd(ctx, workingDir, nil, args...); err != nil {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return errors.Wrap(err, fmt.Sprintf("git push %s %s", upstream, refs))
	}
	return nil
}

// fetch updates refs from the upstream.
func fetch(ctx context.Context, workingDir, upstream string, refspec ...string) error {
	args := append([]string{"fetch", "--tags", upstream}, refspec...)
	if err := execGitCmd(ctx, workingDir, nil, args...); err != nil &&
		!strings.Contains(err.Error(), "Couldn't find remote ref") {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return errors.Wrap(err, fmt.Sprintf("git fetch --tags %s %s", upstream, refspec))
	}
	return nil
}

func refExists(ctx context.Context, workingDir, ref string) (bool, error) {
	if err := execGitCmd(ctx, workingDir, nil, "rev-list", ref); err != nil {
		if strings.Contains(err.Error(), "unknown revision") {
			return false, nil
		}
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return false, err
	}
	return true, nil
}

// Get the full ref for a shorthand notes ref.
func getNotesRef(ctx context.Context, workingDir, ref string) (string, error) {
	out := &bytes.Buffer{}
	if err := execGitCmd(ctx, workingDir, out, "notes", "--ref", ref, "get-ref"); err != nil {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func addNote(ctx context.Context, workingDir, rev, notesRef string, note interface{}) error {
	b, err := json.Marshal(note)
	if err != nil {
		return err
	}
	if err := execGitCmd(ctx, workingDir, nil, "notes", "--ref", notesRef, "add", "-m", string(b), rev); err != nil {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return err
	}
	return nil
}

func getNote(ctx context.Context, workingDir, notesRef, rev string, note interface{}) (ok bool, err error) {
	out := &bytes.Buffer{}
	if err := execGitCmd(ctx, workingDir, out, "notes", "--ref", notesRef, "show", rev); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no note found for object") {
			return false, nil
		}
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return false, err
	}
	if err := json.NewDecoder(out).Decode(note); err != nil {
		return false, err
	}
	return true, nil
}

// Get all revisions with a note (NB: DO NOT RELY ON THE ORDERING)
// It appears to be ordered by ascending git object ref, not by time.
// Return a map to make it easier to do "if in" type queries.
func noteRevList(ctx context.Context, workingDir, notesRef string) (map[string]struct{}, error) {
	out := &bytes.Buffer{}
	if err := execGitCmd(ctx, workingDir, out, "notes", "--ref", notesRef, "list"); err != nil {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return nil, err
	}
	noteList := splitList(out.String())
	result := make(map[string]struct{}, len(noteList))
	for _, l := range noteList {
		split := strings.Fields(l)
		if len(split) > 0 {
			result[split[1]] = struct{}{} // First field contains the object ref (commit id in our case)
		}
	}
	return result, nil
}

// Get the commit hash for a reference
// 获取某一分支的提交commit, ref变量指定分支名称
func refRevision(ctx context.Context, path, ref string) (string, error) {
	out := &bytes.Buffer{}
	if err := execGitCmd(ctx, path, out, "rev-list", "--max-count", "1", ref); err != nil {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func revlist(ctx context.Context, path, ref string) ([]string, error) {
	out := &bytes.Buffer{}
	if err := execGitCmd(ctx, path, out, "rev-list", ref); err != nil {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return nil, err
	}
	return splitList(out.String()), nil
}

// Return the revisions and one-line log commit messages
// subdir argument ... corresponds to the git-path flag supplied to weave-flux-agent
func onelinelog(ctx context.Context, path, refspec, subdir string) ([]Commit, error) {
	out := &bytes.Buffer{}

	// we need to distinguish whether subdir is populated or not,
	// because supplying an empty string to execGitCmd results in git complaining about
	// >> ambiguous argument '' <<
	if subdir != "" {
		if err := execGitCmd(ctx, path, out, "log", "--oneline", "--no-abbrev-commit", refspec, "--", subdir); err != nil {
			buf := make([]byte, 1024)
			outPut := make([]byte, 1024+len(err.Error()))
			strReader := strings.NewReader(err.Error() + "\n")
			strReader.Read(outPut)
			for {
				n := runtime.Stack(buf, false)
				if n < len(buf) {
					outPut = append(outPut, buf...)
					glog.Info(string(outPut))
					break
				}
				buf = make([]byte, 2*len(buf))
			}
			return nil, err
		}
		return splitLog(out.String())
	}

	if err := execGitCmd(ctx, path, out, "log", "--oneline", "--no-abbrev-commit", refspec); err != nil {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return nil, err
	}

	return splitLog(out.String())
}

func splitLog(s string) ([]Commit, error) {
	lines := splitList(s)
	commits := make([]Commit, len(lines))
	for i, m := range lines {
		revAndMessage := strings.SplitN(m, " ", 2)
		commits[i].Revision = revAndMessage[0]
		commits[i].Message = revAndMessage[1]
	}
	return commits, nil
}

func splitList(s string) []string {
	outStr := strings.TrimSpace(s)
	if outStr == "" {
		return []string{}
	}
	return strings.Split(outStr, "\n")
}

// Move the tag to the ref given and push that tag upstream
func moveTagAndPush(ctx context.Context, path string, tag, ref, msg, upstream string) error {
	if err := execGitCmd(ctx, path, nil, "tag", "--force", "-a", "-m", msg, tag, ref); err != nil {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return errors.Wrap(err, "moving tag "+tag)
	}
	if err := execGitCmd(ctx, path, nil, "push", "--force", upstream, "tag", tag); err != nil {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return errors.Wrap(err, "pushing tag to origin")
	}
	return nil
}

func changedFiles(ctx context.Context, path, subPath, ref string) ([]string, error) {
	// Remove leading slash if present. diff doesn't work when using github style root paths.
	if len(subPath) > 0 && subPath[0] == '/' {
		return []string{}, errors.New("git subdirectory should not have leading forward slash")
	}
	out := &bytes.Buffer{}
	// This uses --diff-filter to only look at changes for file _in
	// the working dir_; i.e, we do not report on things that no
	// longer appear.
	if err := execGitCmd(ctx, path, out, "diff", "--name-only", "--diff-filter=ACMRT", ref, "--", subPath); err != nil {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return nil, err
	}
	return splitList(out.String()), nil
}

func fileLastCommit(ctx context.Context, path, subPath, file string) (string, error) {
	out := &bytes.Buffer{}
	err := execGitCmd(ctx, path, out, "log", "-n", "1", "--pretty=format:%H", "--", file)
	if err != nil {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		return "", err
	}
	return out.String(), nil
}

func listToString(files []string) string {
	result := ""
	for _, file := range files {
		result = result + file + " "
	}
	return result
}

func filesPreviousCommit(ctx context.Context, path, subPath, ref string, files []string) (map[string]string, error) {
	result := map[string]string{}
	if len(subPath) > 0 && subPath[0] == '/' {
		return result, errors.New("git subdirectory should not have leading forward slash")
	}
	out := &bytes.Buffer{}
	if err := execGitCmd(ctx, path, out, "log", "-n", "1", "--pretty=format:%h --", listToString(files)); err != nil {
		buf := make([]byte, 1024)
		outPut := make([]byte, 1024+len(err.Error()))
		strReader := strings.NewReader(err.Error() + "\n")
		strReader.Read(outPut)
		for {
			n := runtime.Stack(buf, false)
			if n < len(buf) {
				outPut = append(outPut, buf...)
				glog.Info(string(outPut))
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		log.Fatal(out.String())
	}
	return nil, nil
}

func execGitCmd(ctx context.Context, dir string, out io.Writer, args ...string) error {
	if trace {
		print("TRACE: git")
		for _, arg := range args {
			print(` "`, arg, `"`)
		}
		println()
	}
	glog.V(2).Info(args)
	c := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		c.Dir = dir
	}
	c.Env = env()
	c.Stdout = ioutil.Discard
	if out != nil {
		c.Stdout = out
	}
	errOut := &bytes.Buffer{}
	c.Stderr = errOut
	err := c.Run()
	if err != nil {
		msg := findErrorMessage(errOut)
		if msg != "" {
			err = errors.New(msg)
		}
	}
	if ctx.Err() == context.DeadlineExceeded {
		return errors.Wrap(ctx.Err(), fmt.Sprintf("running git command: %s %v", "git", args))
	} else if ctx.Err() == context.Canceled {
		return errors.Wrap(ctx.Err(), fmt.Sprintf("context was unexpectedly cancelled when running git command: %s %v", "git", args))
	}
	return err
}

func env() []string {
	return []string{"GIT_TERMINAL_PROMPT=0"}
}

// check returns true if there are changes locally.
func check(ctx context.Context, workingDir, subdir string) bool {
	// `--quiet` means "exit with 1 if there are changes"
	return execGitCmd(ctx, workingDir, nil, "diff", "--quiet", "--", subdir) != nil
}

func findErrorMessage(output io.Reader) string {
	sc := bufio.NewScanner(output)
	for sc.Scan() {
		switch {
		case strings.HasPrefix(sc.Text(), "fatal: "):
			return sc.Text()
		case strings.HasPrefix(sc.Text(), "ERROR fatal: "): // Saw this error on ubuntu systems
			return sc.Text()
		case strings.HasPrefix(sc.Text(), "error:"):
			return strings.Trim(sc.Text(), "error: ")
		}
	}
	return ""
}
