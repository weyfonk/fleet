#!/bin/bash
#
# Submit new Fleet version against rancher/charts

set -ue

PREV_FLEET_VERSION="$1"   # e.g. 0.5.2-rc.3
NEW_FLEET_VERSION="$2"
PREV_CHART_VERSION="$3"   # e.g. 101.2.0
NEW_CHART_VERSION="$4"
REPLACE="$5"              # remove previous version if `true`, otherwise add new
CHART_URL="$6" # source chart from a different URL for testing purposes

CHARTS_DIR=${CHARTS_DIR-"$(dirname -- "$0")/../../../charts"}

pushd "${CHARTS_DIR}" > /dev/null

if [ ! -e ~/.gitconfig ]; then
    git config --global user.name "fleet-bot"
    git config --global user.email fleet@suse.de
fi

if [ ! -f bin/charts-build-scripts ]; then
    make pull-scripts
fi

if grep -q "version: ${PREV_CHART_VERSION}" ./packages/fleet/fleet/package.yaml && grep -q "${PREV_FLEET_VERSION}" ./packages/fleet/fleet/package.yaml; then

    find ./packages/fleet/ -type f -exec sed -i -e "s/${PREV_FLEET_VERSION}/${NEW_FLEET_VERSION}/g" {} \;
    find ./packages/fleet/ -type f -exec sed -i -e "s/version: ${PREV_CHART_VERSION}/version: ${NEW_CHART_VERSION}/g" {} \;

    if [[ $# -eq 6 && -n $CHART_URL ]]; then
        echo "### Custom chart URL: $CHART_URL"
        find ./packages/fleet/ -type f -name 'package.yaml' -exec sed -i -e "s,^url:.*${NEW_FLEET_VERSION},url: $CHART_URL," {} \;
        # DEBUG
        find ./packages/fleet/ -type f -name 'package.yaml' -exec cat {} \;
    fi
else
    echo "Previous Fleet version references do not exist in ./packages/fleet/ so replacing it with the new version is not possible. Exiting..."
    exit 1
fi

for i in fleet fleet-crd fleet-agent; do
    yq --inplace "del( .${i}.[] | select(. == \"${PREV_CHART_VERSION}+up${PREV_FLEET_VERSION}\") )" release.yaml
    yq --inplace ".${i} += [\"${NEW_CHART_VERSION}+up${NEW_FLEET_VERSION}\"]" release.yaml
done

# Adapt Gitjob version in generated patch
if grep -q '^-  version: ' ./packages/fleet/fleet/generated-changes/patch/Chart.yaml.patch; then
    GITJOB_VERSION=$(curl -s "https://raw.githubusercontent.com/rancher/fleet/v${NEW_FLEET_VERSION}/charts/fleet/charts/gitjob/Chart.yaml" | yq e .version)
    sed -i -e "s/^-  version: .*$/-  version: ${GITJOB_VERSION}/" ./packages/fleet/fleet/generated-changes/patch/Chart.yaml.patch
fi

git add packages/fleet release.yaml
git commit -m "Updating to Fleet v${NEW_FLEET_VERSION}"

if [ "${REPLACE}" == "true" ]; then
    for i in fleet fleet-crd fleet-agent; do
        CHART=$i VERSION=${PREV_CHART_VERSION}+up${PREV_FLEET_VERSION} make remove
    done
fi

PACKAGE=fleet make charts
git add assets/fleet* charts/fleet* index.yaml
git commit -m "Autogenerated changes for Fleet v${NEW_FLEET_VERSION}"

popd > /dev/null
