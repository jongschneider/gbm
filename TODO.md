2. ✅ gbm2 wt list (TUI)
    * ✅ if a worktree is tracked, remove the option to push (illegal activity)

3. ✅ gbm2 wt push
    * ✅ print out the git operation output with the link to make the PR
    <other-version>
```sh
  gbm push
💡 Pushing current worktree 'HOTFIX_INGSVC-6476'...
Enumerating objects: 17, done.
Counting objects: 100% (17/17), done.
Delta compression using up to 10 threads
Compressing objects: 100% (9/9), done.
Writing objects: 100% (9/9), 4.03 MiB | 7.35 MiB/s, done.
Total 9 (delta 7), reused 0 (delta 0), pack-reused 0 (from 0)
remote:
remote: Create pull request for hotfix/INGSVC-6476-ms-copilot-fix-test_to_FIGHTCLUB:
remote:   https://bitbucket.org/thetalake/integrator/pull-requests/new?source=hotfix/INGSVC-6476-ms-copilot-fix-test_to_FIGHTCLUB&t=1
remote:
To bitbucket.org:thetalake/integrator.git
 * [new branch]          hotfix/INGSVC-6476-ms-copilot-fix-test_to_FIGHTCLUB -> hotfix/INGSVC-6476-ms-copilot-fix-test_to_FIGHTCLUB
branch 'hotfix/INGSVC-6476-ms-copilot-fix-test_to_FIGHTCLUB' set up to track 'origin/hotfix/INGSVC-6476-ms-copilot-fix-test_to_FIGHTCLUB'.
    </other-version>

4. gbm2 wt add (TUI and non-TUI)
    * create a jira ticket md file with all of the jira ticket information (summary, etc)
    * we should check what info we are getting from the jira cli to determine the format of this doc

5. ✅ gbm2 wt add (TUI and non-TUI)
    * ✅ add base branch to ouput
    <current-output>
```sh
  gbm2 wt add

✓ Worktree created successfully!
  Name:   testing123
  Path:   /Users/jschneider/code/scratch/integrator/worktrees/testing123
  Branch: feature/testing123
  Commit: 2b0efde2e
```
    </current-output>