import QtQuick
import QtQuick.Layouts
import org.kde.kirigami as Kirigami
import org.kde.plasma.components as PlasmaComponents
import org.kde.plasma.plasmoid
import org.kde.plasma.plasma5support as Plasma5Support
import "Usage.js" as Usage

PlasmoidItem {
    id: root
    property var usage: null
    property string refreshError: ""
    property bool fetching: false
    property bool manualRefresh: false
    property double now: Date.now()
    property double lastUpdated: 0
    readonly property var panelRemaining: usage && usage.fiveHour ? usage.fiveHour.remaining : null

    function percentageColor(remaining, normalColor) {
        if (typeof remaining !== "number")
            return normalColor;
        if (remaining < 10)
            return "#da4453";
        if (remaining < 30)
            return "#fdbc4b";
        return normalColor;
    }

    function quotaText(key, label) {
        var quota = usage ? usage[key] : null;
        return label + ": " + (quota && typeof quota.remaining === "number"
            ? Math.round(quota.remaining) + "% left" : i18n("Unavailable"));
    }

    hideOnWindowDeactivate: !Plasmoid.configuration.pinned
    toolTipMainText: "Codex Usage"
    toolTipSubText: quotaText("fiveHour", "5-hour") + "\n" + quotaText("weekly", "Weekly")
        + (refreshError ? "\n" + refreshError : "")

    function refresh(manual) {
        if (fetching)
            return;
        manualRefresh = manual === true;
        fetching = true;
        source.connectSource('"$HOME/.local/bin/codex-usage"');
    }

    Plasma5Support.DataSource {
        id: source
        engine: "executable"
        connectedSources: []
        onNewData: function(sourceName, data) {
            disconnectSource(sourceName);
            root.fetching = false;
            if (data["exit code"] !== 0) {
                root.refreshError = String(data.stderr || i18n("Unable to refresh usage")).trim();
                return;
            }
            try {
                var result = JSON.parse(data.stdout);
                if (!result || !("fiveHour" in result) || !("weekly" in result))
                    throw new Error("Invalid usage response");
                root.usage = result;
                root.lastUpdated = Date.now();
                root.now = root.lastUpdated;
                root.refreshError = "";
            } catch (error) {
                root.refreshError = i18n("Invalid usage response: %1", error.message);
            }
        }
    }

    Timer {
        interval: typeof root.panelRemaining === "number" && root.panelRemaining > 0 && root.panelRemaining <= 30 ? 20000 : 60000
        running: true
        repeat: true
        onTriggered: root.refresh()
    }
    Timer {
        interval: 1000
        running: root.expanded
        repeat: true
        onTriggered: root.now = Date.now()
    }
    Component.onCompleted: refresh()
    onExpandedChanged: {
        now = Date.now();
        if (!expanded)
            Plasmoid.configuration.pinned = false;
    }
    compactRepresentation: PlasmaComponents.ToolButton {
        id: panelButton
        text: "Codex Usage"
        implicitWidth: 32
        implicitHeight: 32
        padding: 0
        focusPolicy: Qt.TabFocus
        onClicked: root.expanded = !root.expanded

        // Show a border for keyboard navigation only, without a hover outline.
        background: Rectangle {
            color: "transparent"
            border.color: Kirigami.Theme.highlightColor
            border.width: panelButton.visualFocus ? 1 : 0
            radius: 3
        }
        contentItem: Item {
            opacity: root.refreshError ? 0.5 : 1
            PlasmaComponents.Label {
                anchors.fill: parent
                color: root.percentageColor(root.panelRemaining, Kirigami.Theme.textColor)
                text: typeof root.panelRemaining === "number" ? Math.round(root.panelRemaining) + "%" : "—"
                horizontalAlignment: Text.AlignHCenter
                verticalAlignment: Text.AlignVCenter
                font.bold: false
                font.pixelSize: Plasmoid.configuration.textSize
                fontSizeMode: Text.Fit
                minimumPixelSize: 8
            }
            PlasmaComponents.BusyIndicator {
                anchors.fill: parent
                visible: root.fetching && root.manualRefresh
                running: visible
            }
        }
        MouseArea {
            anchors.fill: parent
            acceptedButtons: Qt.MiddleButton
            onClicked: root.refresh(true)
        }
    }

    fullRepresentation: Item {
        Layout.minimumWidth: 300
        Layout.preferredWidth: 300
        Layout.maximumWidth: 300
        Layout.minimumHeight: Math.max(160, content.implicitHeight + 16)
        Layout.preferredHeight: Layout.minimumHeight
        Layout.maximumHeight: Layout.minimumHeight

        ColumnLayout {
            id: content
            anchors.fill: parent
            anchors.margins: 8
            spacing: 4

            RowLayout {
                Layout.fillWidth: true
                Layout.maximumHeight: implicitHeight
                ColumnLayout {
                    Layout.fillWidth: true
                    Layout.maximumHeight: implicitHeight
                    spacing: 2
                    PlasmaComponents.Label {
                        Layout.fillWidth: true
                        text: "Codex Usage"
                        font.bold: true
                    }
                    PlasmaComponents.Label {
                        Layout.fillWidth: true
                        text: root.fetching ? i18n("Updating…") : Usage.updatedText(root.lastUpdated, root.now)
                        font: Kirigami.Theme.smallFont
                        opacity: 0.7
                    }
                }
                PlasmaComponents.ToolButton {
                    Layout.alignment: Qt.AlignTop
                    icon.name: "configure"
                    text: i18n("Settings")
                    display: PlasmaComponents.AbstractButton.IconOnly
                    implicitHeight: 28
                    implicitWidth: 28
                    padding: 3
                    onClicked: Plasmoid.internalAction("configure").trigger()
                    PlasmaComponents.ToolTip.text: root.refreshError || text
                    PlasmaComponents.ToolTip.visible: hovered
                }
                PlasmaComponents.ToolButton {
                    Layout.alignment: Qt.AlignTop
                    icon.name: "window-pin"
                    checkable: true
                    checked: Plasmoid.configuration.pinned
                    onToggled: Plasmoid.configuration.pinned = checked
                    text: i18n("Keep Open")
                    display: PlasmaComponents.AbstractButton.IconOnly
                    implicitHeight: 28
                    implicitWidth: 28
                    padding: 3
                    PlasmaComponents.ToolTip.text: text
                    PlasmaComponents.ToolTip.visible: hovered
                }
            }

            Repeater {
                model: [ { label: "5-hour", key: "fiveHour" }, { label: "Weekly", key: "weekly" } ]
                ColumnLayout {
                    required property var modelData
                    property var quota: root.usage ? root.usage[modelData.key] : null
                    property bool available: quota !== null && typeof quota.remaining === "number"
                    Layout.fillWidth: true
                    spacing: 2
                    RowLayout {
                        PlasmaComponents.Label {
                            text: modelData.label
                            Layout.fillWidth: true
                        }
                        PlasmaComponents.Label {
                            color: root.percentageColor(available ? quota.remaining : null, Kirigami.Theme.textColor)
                            text: (available ? Math.round(quota.remaining) + "% left" : (root.fetching && !root.usage ? "Loading…" : "Unavailable"))
                                + (root.refreshError && root.usage ? " · stale" : "")
                        }
                    }
                    PlasmaComponents.ProgressBar {
                        Layout.fillWidth: true
                        from: 0
                        to: 100
                        value: available ? quota.remaining : 0
                        opacity: root.refreshError ? 0.5 : 1
                    }
                    PlasmaComponents.Label {
                        text: Usage.resetText(quota ? quota.resetsAt : null, root.now)
                        font: Kirigami.Theme.smallFont
                        opacity: 0.7
                        PlasmaComponents.ToolTip.text: quota && quota.resetsAt > 0
                            ? i18n("Resets on %1", Qt.formatDateTime(new Date(quota.resetsAt * 1000), "dddd HH:mm"))
                            : i18n("Reset time unavailable")
                        PlasmaComponents.ToolTip.visible: resetHover.containsMouse
                        MouseArea {
                            id: resetHover
                            anchors.fill: parent
                            hoverEnabled: true
                            acceptedButtons: Qt.NoButton
                        }
                    }
                }
            }
        }
    }
}
