
This solution is deployed to PyPI via the [Jenkinsfile](../../Jenkinsfile) in this repository. 

Whenever the `main` branch has new changes pushed to the `python-client` directory, Jenkins will attempt to deploy those changes to PyPI!

# Version Management

Before trying to deliver a new version of this package to PyPI, update the `version` in [setup.py](../setup.py) (following "Semantic Versioning" scheme) 
If the version is not updated, the CICD solution will not upload the package to PyPI (and will raise an error).

# Recommended Git workflow

We follow the [GitHub Git Flow](https://guides.github.com/introduction/flow/) for this project.
This couples nicely with the CICD scheme described above.

# Recommended GitHub Repository Settings

It is useful to leverage a "GitHub Branch protection rule" to help enforce our GitHub Flow.
The following are some 'protection rules' that we have turned on for this project's repository:

 * *Require pull request reviews before merging:* `checked`
   * *Required approving reviews:* 1
   * *Require review from Code Owners:* `checked` 
* *Require status checks to pass before merging:* `checked`
  * *Require branches to be up to date before merging:* `checked`
* *Restrict who can push to matching branches:* `checked`
