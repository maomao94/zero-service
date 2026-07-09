package modelxml

import (
	"encoding/xml"
	"io"
)

const xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"

// WriteDeviceModel writes a Device_Model XML document without buffering all rows in memory.
func WriteDeviceModel(w io.Writer, items []DevicePointModel) error {
	return WriteDeviceModelStream(w, func(yield func(DevicePointModel) error) error {
		for _, item := range items {
			if err := yield(item); err != nil {
				return err
			}
		}
		return nil
	})
}

// WriteDeviceModelStream writes a Device_Model XML document from a row producer.
func WriteDeviceModelStream(w io.Writer, items func(yield func(DevicePointModel) error) error) error {
	return writeDocument(w, xml.Name{Local: "Device_Model"}, func(enc *xml.Encoder) error {
		return items(func(item DevicePointModel) error {
			return enc.Encode(item)
		})
	})
}

// WritePatrolDeviceModel writes a PatrolDevice_Model XML document.
func WritePatrolDeviceModel(w io.Writer, items []PatrolDeviceModel) error {
	return WritePatrolDeviceModelStream(w, func(yield func(PatrolDeviceModel) error) error {
		for _, item := range items {
			if err := yield(item); err != nil {
				return err
			}
		}
		return nil
	})
}

// WritePatrolDeviceModelStream writes a PatrolDevice_Model XML document from a row producer.
func WritePatrolDeviceModelStream(w io.Writer, items func(yield func(PatrolDeviceModel) error) error) error {
	return writeDocument(w, xml.Name{Local: "PatrolDevice_Model"}, func(enc *xml.Encoder) error {
		return items(func(item PatrolDeviceModel) error {
			return enc.Encode(item)
		})
	})
}

func writeDocument(w io.Writer, root xml.Name, writeRows func(*xml.Encoder) error) error {
	if _, err := io.WriteString(w, xmlHeader); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.EncodeToken(xml.StartElement{Name: root}); err != nil {
		return err
	}
	if err := writeRows(enc); err != nil {
		return err
	}
	if err := enc.EncodeToken(xml.EndElement{Name: root}); err != nil {
		return err
	}
	if err := enc.Flush(); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}
