import QtQuick
import QtQuick.Layouts
import QtQuick.Controls as Controls
import org.kde.kirigami as Kirigami
import org.kde.kcmutils as KCM

KCM.SimpleKCM {
    title: i18n("General")
    property alias cfg_textSize: textSize.value

    Kirigami.FormLayout {
        RowLayout {
            Kirigami.FormData.label: i18n("Text size:")
            Controls.SpinBox {
                id: textSize
                from: 8
                to: 48
                value: 15
                editable: true
            }
            Controls.Label { text: i18n("px") }
        }
    }
}
