package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dualgit/internal/config"
	"dualgit/internal/gitutil"
	"dualgit/internal/ignore"
	"dualgit/internal/ui"
)

type App struct {
	git gitutil.Runner
}

type runtime struct {
	cfg     config.Config
	state   config.State
	matcher ignore.Matcher
}

func New() *App {
	return &App{git: gitutil.Runner{}}
}

func (a *App) Run(args []string) error {
	if len(args) == 0 {
		return a.printHelp()
	}

	switch args[0] {
	case "init":
		return a.cmdInit()
	case "context":
		if len(args) < 2 {
			return errors.New("usage: dualgit context public|private")
		}
		return a.cmdContext(args[1])
	case "status":
		return a.cmdStatus()
	case "push":
		return a.cmdPush(args[1:])
	case "publish":
		return a.cmdPublish()
	case "hooks":
		if len(args) < 2 {
			return errors.New("usage: dualgit hooks install|check")
		}
		switch args[1] {
		case "install":
			return a.installHooks()
		case "check":
			return a.checkHooks()
		default:
			return errors.New("usage: dualgit hooks install|check")
		}
	case "shell-init":
		if len(args) < 2 {
			return errors.New("usage: dualgit shell-init zsh|bash")
		}
		return a.cmdShellInit(args[1])
	case "_hook":
		if len(args) < 2 {
			return errors.New("usage: dualgit _hook pre-commit|pre-push")
		}
		return a.cmdHook(args[1])
	case "_guard":
		if len(args) < 2 {
			return errors.New("usage: dualgit _guard staged")
		}
		return a.cmdGuard(args[1])
	case "_post_commit_sync":
		return a.cmdPostCommitSync()
	case "_prompt":
		return a.cmdPrompt()
	default:
		return a.printHelp()
	}
}

func (a *App) printHelp() error {
	fmt.Println("dualgit commands:")
	fmt.Println("  init")
	fmt.Println("  context public|private")
	fmt.Println("  status")
	fmt.Println("  push [--all-contexts] [--dry-run]")
	fmt.Println("  publish")
	fmt.Println("  hooks install|check")
	fmt.Println("  shell-init zsh|bash")
	return nil
}

func (a *App) loadRuntime() (runtime, error) {
	cfg, err := config.Load()
	if err != nil {
		return runtime{}, fmt.Errorf("load config: %w", err)
	}
	state, err := config.LoadState()
	if err != nil {
		return runtime{}, fmt.Errorf("load state: %w", err)
	}
	matcher, err := ignore.Load(".publicignore")
	if err != nil {
		return runtime{}, fmt.Errorf("load .publicignore: %w", err)
	}
	return runtime{cfg: cfg, state: state, matcher: matcher}, nil
}

func (a *App) cmdInit() error {
	if err := a.git.EnsureRepo(); err != nil {
		return err
	}

	cfg := config.Default()
	publicURL, err := ui.Prompt("Inserisci l'URL del remote PUBBLICO", "")
	if err != nil {
		return err
	}
	privateURL, err := ui.Prompt("Inserisci l'URL del remote PRIVATO", "")
	if err != nil {
		return err
	}
	if strings.TrimSpace(publicURL) == "" || strings.TrimSpace(privateURL) == "" {
		return errors.New("entrambi i remoti sono obbligatori")
	}
	cfg.PublicRemote.URL = publicURL
	cfg.PrivateRemote.URL = privateURL

	if err := a.git.EnsureRemote(cfg.PublicRemote.Name, cfg.PublicRemote.URL); err != nil {
		return err
	}
	if err := a.git.EnsureRemote(cfg.PrivateRemote.Name, cfg.PrivateRemote.URL); err != nil {
		return err
	}

	if a.git.HasHead() {
		current, err := a.git.CurrentBranch()
		if err != nil {
			return err
		}
		base := "HEAD"
		if current != "" {
			base = current
		}
		if err := a.git.EnsureBranch(cfg.PublicBranch, base); err != nil {
			return err
		}
		if err := a.git.EnsureBranch(cfg.PrivateBranch, cfg.PublicBranch); err != nil {
			return err
		}
	}

	if err := config.Save(cfg); err != nil {
		return err
	}
	if err := config.SaveState(config.DefaultState()); err != nil {
		return err
	}
	if err := ensurePublicIgnore(); err != nil {
		return err
	}
	if err := a.installHooks(); err != nil {
		return err
	}
	fmt.Println("DualGit inizializzato. Contesto attuale: public")
	return nil
}

func ensurePublicIgnore() error {
	_, err := os.Stat(".publicignore")
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	content := "# Files only allowed in private context\n"
	return os.WriteFile(".publicignore", []byte(content), 0o644)
}

func (a *App) cmdContext(target string) error {
	rt, err := a.loadRuntime()
	if err != nil {
		return err
	}
	if target != "public" && target != "private" {
		return errors.New("context must be public|private")
	}
	clean, err := a.git.IsWorkTreeClean()
	if err != nil {
		return err
	}
	if !clean {
		return errors.New("working tree non pulito: fai commit/stash prima di cambiare contesto")
	}

	if target == rt.state.Context {
		fmt.Printf("Contesto gia' impostato su %s\n", target)
		return nil
	}

	if target == "private" {
		if err := a.mergePublicIntoPrivate(rt.cfg); err != nil {
			return err
		}
	}

	branch := rt.cfg.PublicBranch
	if target == "private" {
		branch = rt.cfg.PrivateBranch
	}
	if a.git.BranchExists(branch) {
		if err := a.git.Checkout(branch); err != nil {
			return err
		}
	}

	rt.state.Context = target
	if err := config.SaveState(rt.state); err != nil {
		return err
	}
	fmt.Printf("Contesto cambiato: %s\n", target)
	return nil
}

func (a *App) cmdStatus() error {
	rt, err := a.loadRuntime()
	if err != nil {
		return err
	}
	branch, _ := a.git.CurrentBranch()
	clean, _ := a.git.IsWorkTreeClean()
	fmt.Printf("Context: %s\n", rt.state.Context)
	fmt.Printf("Branch corrente: %s\n", branch)
	fmt.Printf("Public: %s/%s -> %s\n", rt.cfg.PublicRemote.Name, rt.cfg.PublicRemote.Branch, rt.cfg.PublicBranch)
	fmt.Printf("Private: %s/%s -> %s\n", rt.cfg.PrivateRemote.Name, rt.cfg.PrivateRemote.Branch, rt.cfg.PrivateBranch)
	fmt.Printf("Working tree clean: %t\n", clean)
	return nil
}

func (a *App) cmdPush(args []string) error {
	rt, err := a.loadRuntime()
	if err != nil {
		return err
	}
	all := false
	dryRun := false
	for _, arg := range args {
		switch arg {
		case "--all-contexts":
			all = true
		case "--dry-run":
			dryRun = true
		default:
			return fmt.Errorf("flag non riconosciuta: %s", arg)
		}
	}

	if all {
		if err := a.pushBranch(rt.cfg, rt.matcher, rt.cfg.PublicBranch, rt.cfg.PublicRemote.Name, rt.cfg.PublicRemote.Branch, dryRun, true); err != nil {
			return err
		}
		if err := a.pushBranch(rt.cfg, rt.matcher, rt.cfg.PrivateBranch, rt.cfg.PrivateRemote.Name, rt.cfg.PrivateRemote.Branch, dryRun, false); err != nil {
			return err
		}
		fmt.Println("Push completato su entrambi i contesti")
		return nil
	}

	if rt.state.Context == "public" {
		if err := a.pushBranch(rt.cfg, rt.matcher, rt.cfg.PublicBranch, rt.cfg.PublicRemote.Name, rt.cfg.PublicRemote.Branch, dryRun, true); err != nil {
			return err
		}
		fmt.Println("Push pubblico completato")
		return nil
	}

	if err := a.pushBranch(rt.cfg, rt.matcher, rt.cfg.PrivateBranch, rt.cfg.PrivateRemote.Name, rt.cfg.PrivateRemote.Branch, dryRun, false); err != nil {
		return err
	}
	fmt.Println("Push privato completato")
	return nil
}

func (a *App) pushBranch(cfg config.Config, matcher ignore.Matcher, localBranch, remote, remoteBranch string, dryRun, validatePublic bool) error {
	if validatePublic {
		if err := a.ensureNoPublicLeaks(localBranch, remote, remoteBranch, matcher); err != nil {
			return err
		}
	}
	args := []string{"push"}
	if dryRun {
		args = append(args, "--dry-run")
	}
	args = append(args, remote, fmt.Sprintf("%s:%s", localBranch, remoteBranch))
	return a.git.RunEnv(map[string]string{"DUALGIT_ALLOW_PUSH": "1"}, args...)
}

func (a *App) ensureNoPublicLeaks(localBranch, remote, remoteBranch string, matcher ignore.Matcher) error {
	remoteRef := fmt.Sprintf("refs/remotes/%s/%s", remote, remoteBranch)
	_, err := a.git.Output("show-ref", "--verify", "--quiet", remoteRef)
	hasRemoteRef := err == nil
	rangeSpec := localBranch
	if hasRemoteRef {
		rangeSpec = fmt.Sprintf("%s..%s", remoteRef, localBranch)
	}
	commits, err := a.git.OutputLines("rev-list", rangeSpec)
	if err != nil {
		return err
	}
	for _, c := range commits {
		files, err := a.git.OutputLines("diff-tree", "--no-commit-id", "--name-only", "-r", c)
		if err != nil {
			return err
		}
		for _, f := range files {
			if matcher.Match(f) {
				return fmt.Errorf("leak rilevato nel commit %s: %s e' protetto da .publicignore", shortSHA(c), f)
			}
		}
	}
	return nil
}

func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

func (a *App) cmdPublish() error {
	rt, err := a.loadRuntime()
	if err != nil {
		return err
	}
	clean, err := a.git.IsWorkTreeClean()
	if err != nil {
		return err
	}
	if !clean {
		return errors.New("working tree non pulito: publish richiede stato pulito")
	}

	lines, err := a.git.OutputLines("log", "--reverse", "--pretty=format:%H\t%s", rt.cfg.PublicBranch+".."+rt.cfg.PrivateBranch)
	if err != nil {
		return err
	}
	if len(lines) == 0 {
		fmt.Println("Nessun commit privato da pubblicare")
		return nil
	}
	fmt.Println("Commit privati candidati:")
	for i, l := range lines {
		parts := strings.SplitN(l, "\t", 2)
		subj := ""
		if len(parts) == 2 {
			subj = parts[1]
		}
		fmt.Printf("  %d) %s %s\n", i+1, shortSHA(parts[0]), subj)
	}

	selected, err := ui.SelectMany(len(lines))
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		fmt.Println("Nessun commit selezionato")
		return nil
	}

	orig, _ := a.git.CurrentBranch()
	if orig != rt.cfg.PublicBranch {
		if err := a.git.Checkout(rt.cfg.PublicBranch); err != nil {
			return err
		}
		defer func() { _ = a.git.Checkout(orig) }()
	}

	for _, idx := range selected {
		parts := strings.SplitN(lines[idx-1], "\t", 2)
		sha := parts[0]
		subject := sha
		if len(parts) == 2 {
			subject = parts[1]
		}
		if err := a.git.Run("cherry-pick", "--no-commit", sha); err != nil {
			_ = a.git.Run("cherry-pick", "--abort")
			return fmt.Errorf("conflitto durante publish su %s: risolvi manualmente e riprova", shortSHA(sha))
		}
		if err := a.filterIgnoredFromStage(rt.matcher); err != nil {
			_ = a.git.Run("cherry-pick", "--abort")
			return err
		}

		staged, err := a.git.Output("diff", "--cached", "--name-only")
		if err != nil {
			return err
		}
		if strings.TrimSpace(staged) == "" {
			if err := a.git.Run("reset", "--hard", "HEAD"); err != nil {
				return err
			}
			fmt.Printf("Skip %s: solo file privati\n", shortSHA(sha))
			continue
		}
		if err := a.git.Run("commit", "--no-verify", "-m", subject, "-m", "DualGit-Published-From: "+sha); err != nil {
			return err
		}
		fmt.Printf("Pubblicato %s\n", shortSHA(sha))
	}

	fmt.Println("Publish completato")
	return nil
}

func (a *App) filterIgnoredFromStage(matcher ignore.Matcher) error {
	staged, err := a.git.OutputLines("diff", "--cached", "--name-only")
	if err != nil {
		return err
	}
	for _, path := range staged {
		if matcher.Match(path) {
			if err := a.git.Run("restore", "--staged", "--worktree", "--", path); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *App) installHooks() error {
	if err := os.MkdirAll(filepath.Join(".git", "hooks"), 0o755); err != nil {
		return err
	}
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	qexe := shellQuote(exePath)
	preCommit := "#!/bin/sh\nexec " + qexe + " _hook pre-commit\n"
	prePush := "#!/bin/sh\nexec " + qexe + " _hook pre-push \"$@\"\n"
	if err := os.WriteFile(filepath.Join(".git", "hooks", "pre-commit"), []byte(preCommit), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(".git", "hooks", "pre-push"), []byte(prePush), 0o755); err != nil {
		return err
	}
	fmt.Println("Hooks installati")
	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func (a *App) checkHooks() error {
	for _, name := range []string{"pre-commit", "pre-push"} {
		path := filepath.Join(".git", "hooks", name)
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("hook mancante: %s", name)
		}
	}
	fmt.Println("Hooks OK")
	return nil
}

func (a *App) cmdShellInit(shell string) error {
	switch shell {
	case "bash":
		fmt.Printf("%s", bashInit)
		return nil
	case "zsh":
		fmt.Printf("%s", zshInit)
		return nil
	default:
		return errors.New("shell supportata: zsh|bash")
	}
}

func (a *App) cmdHook(name string) error {
	switch name {
	case "pre-commit":
		return a.cmdGuard("staged")
	case "pre-push":
		if os.Getenv("DUALGIT_ALLOW_PUSH") == "1" {
			return nil
		}
		return errors.New("push bloccato: usa 'dualgit push'")
	default:
		return errors.New("hook non supportato")
	}
}

func (a *App) cmdGuard(scope string) error {
	if scope != "staged" {
		return errors.New("guard scope non supportato")
	}
	rt, err := a.loadRuntime()
	if err != nil {
		return err
	}
	if rt.state.Context != "public" {
		return nil
	}
	staged, err := a.git.OutputLines("diff", "--cached", "--name-only")
	if err != nil {
		return err
	}
	violations := make([]string, 0)
	for _, p := range staged {
		if rt.matcher.Match(p) {
			violations = append(violations, p)
		}
	}
	if len(violations) == 0 {
		return nil
	}
	for _, p := range violations {
		fmt.Printf("- %s\n", p)
	}
	return errors.New("commit bloccato: file protetti da .publicignore nel contesto public")
}

func (a *App) cmdPostCommitSync() error {
	rt, err := a.loadRuntime()
	if err != nil {
		return err
	}
	if rt.state.Context != "public" {
		return nil
	}
	if err := a.syncPublicCommitsToPrivate(rt.cfg); err != nil {
		return fmt.Errorf("sync public->private fallita: %w", err)
	}
	return nil
}

func (a *App) cmdPrompt() error {
	st, err := config.LoadState()
	if err != nil {
		return nil
	}
	fmt.Print(st.Context)
	return nil
}

func (a *App) syncPublicCommitsToPrivate(cfg config.Config) error {
	clean, err := a.git.IsWorkTreeClean()
	if err != nil {
		return err
	}
	if !clean {
		return errors.New("working tree sporco, impossibile sincronizzare")
	}
	orig, _ := a.git.CurrentBranch()
	commits, err := a.git.OutputLines("rev-list", "--reverse", cfg.PrivateBranch+".."+cfg.PublicBranch)
	if err != nil {
		return err
	}
	if len(commits) == 0 {
		return nil
	}
	if err := a.git.Checkout(cfg.PrivateBranch); err != nil {
		return err
	}
	for _, c := range commits {
		if err := a.git.Run("cherry-pick", c); err != nil {
			_ = a.git.Run("cherry-pick", "--abort")
			if orig != "" {
				_ = a.git.Checkout(orig)
			}
			return fmt.Errorf("conflitto su %s", shortSHA(c))
		}
	}
	if orig != "" {
		if err := a.git.Checkout(orig); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) mergePublicIntoPrivate(cfg config.Config) error {
	orig, _ := a.git.CurrentBranch()
	if err := a.git.Checkout(cfg.PrivateBranch); err != nil {
		return err
	}
	if err := a.git.Run("merge", "--no-edit", cfg.PublicBranch); err != nil {
		_ = a.git.Run("merge", "--abort")
		if orig != "" {
			_ = a.git.Checkout(orig)
		}
		return errors.New("merge public->private fallita, risolvi i conflitti manualmente")
	}
	if orig != "" {
		if err := a.git.Checkout(orig); err != nil {
			return err
		}
	}
	return nil
}

const bashInit = `# dualgit shell init (bash)
git() {
  if [ "$1" = "commit" ]; then
    command git "$@"
    rc=$?
    if [ $rc -eq 0 ]; then
      dualgit _post_commit_sync || return $?
    fi
    return $rc
  fi
  if [ "$1" = "push" ]; then
    shift
    dualgit push "$@"
    return $?
  fi
  command git "$@"
  rc=$?
  if [ $rc -eq 0 ] && [ "$1" = "add" ]; then
    dualgit _guard staged || return $?
  fi
  return $rc
}

_dualgit_prompt_prefix() {
  local ctx
  ctx="$(dualgit _prompt 2>/dev/null)"
  if [ -n "$ctx" ]; then
    printf "(%s) " "$ctx"
  fi
}
PS1='$(_dualgit_prompt_prefix)'"$PS1"
`

const zshInit = `# dualgit shell init (zsh)
git() {
  if [ "$1" = "commit" ]; then
    command git "$@"
    local rc=$?
    if [ $rc -eq 0 ]; then
      dualgit _post_commit_sync || return $?
    fi
    return $rc
  fi
  if [ "$1" = "push" ]; then
    shift
    dualgit push "$@"
    return $?
  fi
  command git "$@"
  local rc=$?
  if [ $rc -eq 0 ] && [ "$1" = "add" ]; then
    dualgit _guard staged || return $?
  fi
  return $rc
}

dualgit_precmd() {
  local ctx
  ctx="$(dualgit _prompt 2>/dev/null)"
  if [ -n "$ctx" ]; then
    DUALGIT_PROMPT_PREFIX="($ctx) "
  else
    DUALGIT_PROMPT_PREFIX=""
  fi
}
autoload -Uz add-zsh-hook 2>/dev/null || true
add-zsh-hook precmd dualgit_precmd 2>/dev/null || precmd_functions+=(dualgit_precmd)
PROMPT='${DUALGIT_PROMPT_PREFIX}'"$PROMPT"
`
