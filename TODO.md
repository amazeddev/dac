DAC (Data Analisys Configurator)

- flag in chain config weather result should be saved if is linked or removed/overwritten
- ~~mechanism to save result of procesing in hdf5~~
- more python functions
- support for python custom functions
- put all data connected python files in one big class linke in chains.py
- helper function calculating and storing column info together with column
- make it as a repo with ci/cd
- download **_.dac_** data (libs, kvstore executable) on first app start (new instalation)
- many processings (procesing as list od objects witch chains, name, etc.) each save own results, and could be run separately

CONNECTOR

- ~~get `interface{}` as value, not chain struct~~
- make it as repo with ci/cd and downloadable binary
- move client to this repo and fetch it in DAC

LIBS

- put data in repo, set ci/cd for zipping, tagging and publishing like [here](https://keithweaverca.medium.com/zip-code-base-with-github-actions-for-releases-aca66f530dae) ([zip/release actions](https://github.com/marketplace/actions/zip-release))
