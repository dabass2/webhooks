# Github Webhook Handler (go[lang] version)

This is a github webhook handler written in the GO (go) language (golang). Uses gin (the go [golang] framework) to do the whole web request thing and response and stuff.

I have never used go(lang) before so I'm pretty sure it's likely 100% correct and well written.

Basically it works by running a web server on whatever port you tell it to. Then you set this up in your webhooks section and have it send the push event (meaning this code will run whenever any push occurs to the specified branch). This should be done for the pull request event, but I do not want to do that right now. Anyway, then you'll need a `projects.json` file setup. I had a desc typed up here, but just look at this code snippet

```
{
  "projects": [
    {
      "repoName": "test",
      "acceptedBranches": ["master"],
      "scriptName": "test.sh",
      "desc": "Test project to test this webhook server."
    }
  ]
}
```

The `repoName` and `acceptedBranches` must contain what repo and branch you want this code to run on. Then the `scriptName` must be the name of the script you want to run. It's gotta be a bash script though hehe.

In order to run this it is required to have a .env file (EXCEPT IT'S NOT BC THERE'S DEFAULTS). The .env file should be in the root dir with the name `.env` and contains the following information

`.env`

| Variable Name      | Default Value | Type   |
| ------------------ | ------------- | ------ |
| GIN_ADDR           | 0.0.0.0       | String |
| GIN_PORT           | 8025          | String |
| CHECK_GITHUB_HASH  | False         | Bool   |
| PRJ_FILE_DIR       | ./            | String |
| PRJ_FILE_NAME      | projects.json | String |
| GITHUB_SECRET      | N/A           | String |

This README easily could be better, but I don't want to.