DAC (Data Analisys Configurator)

- ~~fetch on run (check if all necessary columns are fetched from import table)~~
- flag in chain config weather result should be saved if is linked or removed/overwritten (right now autoremoved if used)
- ~~mechanism to save result of procesing in hdf5~~
- more python functions
- support for python custom functions
- ~~support for python lambda functions~~
- ~~put all data connected python files in one big class linke in chains.py~~
- helper function calculating and storing column info together with column
- make it as a repo with ci/cd
- ~~be able to create python virtual env dor _dac_ to use~~
- ~~download **_.dac_** data (libs, kvstore executable) on first app start (new instalation) and create venv~~
- many processings/workflows (procesing as list od objects witch chains, name, etc.) each save own results, and could be run separately
- SQL import sypport
- ~~name, import etc. specified on init (interacrive prompts)~~
- ~~when _dac run -f_ have to check if all columns that should be removed are ~~
- possible group of targets for each chain (same prosessing executed on each column of group)
- command to destroy project (clear everything)

CONNECTOR

- ~~get `interface{}` as value, not chain struct~~
- make it as repo with ci/cd and downloadable binary
- move client to this repo and fetch it in DAC

LIBS

- put data in repo, set ci/cd for zipping, tagging and publishing like [here](https://keithweaverca.medium.com/zip-code-base-with-github-actions-for-releases-aca66f530dae) ([zip/release actions](https://github.com/marketplace/actions/zip-release))

export PATH=$PATH:$HOME/go/bin
cp -R ./lib /home/sebastianluszczek/.dac/ (cp -R ./lib /home/sebastianluszczek/.dac/ && cp -R ./requirements.txt /home/sebastianluszczek/.dac/)
go build -o ~/.dac/kvstore main.go
