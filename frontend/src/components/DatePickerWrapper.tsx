'use client';

import React from 'react';
import DatePicker from 'react-datepicker';
import 'react-datepicker/dist/react-datepicker.css';
import { zhCN } from 'date-fns/locale';

export interface DatePickerWrapperProps {
  selected: Date | null;
  onChange: (date: Date | null) => void;
  dateFormat?: string;
  className?: string;
  placeholderText?: string;
  showYearDropdown?: boolean;
  showMonthDropdown?: boolean;
  dropdownMode?: 'scroll' | 'select';
  maxDate?: Date;
  minDate?: Date;
  label?: string;
}

export function DatePickerWrapper(props: DatePickerWrapperProps) {
  const {
    selected,
    onChange,
    dateFormat,
    className,
    placeholderText,
    showYearDropdown,
    showMonthDropdown,
    dropdownMode,
    maxDate,
    minDate,
    label = '选择日期',
    ...rest
  } = props;

  return (
    <div>
      {label && <label className="sr-only">{label}</label>}
      <DatePicker 
        selected={selected}
        onChange={onChange}
        dateFormat={dateFormat}
        className={className}
        placeholderText={placeholderText}
        showYearDropdown={showYearDropdown}
        showMonthDropdown={showMonthDropdown}
        dropdownMode={dropdownMode}
        maxDate={maxDate}
        minDate={minDate}
        locale={zhCN}
        yearDropdownItemNumber={20}
        {...rest}
      />
    </div>
  );
}
