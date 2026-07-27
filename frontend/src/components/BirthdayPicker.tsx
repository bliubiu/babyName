'use client';

import { useState } from 'react';
import DatePicker from 'react-datepicker';
import 'react-datepicker/dist/react-datepicker.css';
import { zhCN } from 'date-fns/locale';

interface BirthdayPickerProps {
  onConfirm: (data: {
    birthYear: number;
    birthMonth: number;
    birthDay: number;
    birthHour: number;
    birthMinute: number;
    birthType: 'solar' | 'lunar';
  }) => void;
  initialDisplay?: string;
}

function formatDate(date: Date): string {
  const y = date.getFullYear();
  const M = (date.getMonth() + 1).toString().padStart(2, '0');
  const d = date.getDate().toString().padStart(2, '0');
  const h = date.getHours().toString().padStart(2, '0');
  const m = date.getMinutes().toString().padStart(2, '0');
  return `${y}-${M}-${d} ${h}:${m}`;
}

export function BirthdayPicker({ onConfirm, initialDisplay = '请选择出生时间' }: BirthdayPickerProps) {
  const [showModal, setShowModal] = useState(false);
  const [selectedDate, setSelectedDate] = useState<Date>(new Date());
  const [display, setDisplay] = useState<string>(initialDisplay);

  const openPicker = () => setShowModal(true);
  const closePicker = () => setShowModal(false);

  const confirm = () => {
    if (selectedDate) {
      onConfirm({
        birthYear: selectedDate.getFullYear(),
        birthMonth: selectedDate.getMonth() + 1,
        birthDay: selectedDate.getDate(),
        birthHour: selectedDate.getHours(),
        birthMinute: selectedDate.getMinutes(),
        birthType: 'solar'
      });
      setDisplay(formatDate(selectedDate));
    }
    closePicker();
  };

  return (
    <>
      <button
        type="button"
        onClick={openPicker}
        className="input-field flex-1 max-w-md text-left cursor-pointer"
      >
        {display}
      </button>

      {showModal && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50">
          <div className="bg-warm-white-light rounded-xl w-full max-w-md mx-4 overflow-hidden shadow-xl border border-paper">
            <div className="text-center py-4 border-b border-paper">
              <span className="text-lg font-serif text-ink">选择出生时间</span>
            </div>

            <div className="p-6">
              <div className="text-center text-lg font-serif text-ink mb-4">
                {selectedDate.getFullYear()}年{selectedDate.getMonth() + 1}月{selectedDate.getDate()}日 {selectedDate.getHours()}时{selectedDate.getMinutes()}分
              </div>

              <div className="mb-6">
                <DatePicker
                  selected={selectedDate}
                  onChange={(date: Date | null) => setSelectedDate(date || new Date())}
                  showTimeSelect
                  timeFormat="HH:mm"
                  timeIntervals={15}
                  dateFormat="yyyy-MM-dd HH:mm"
                  className="input-field"
                  locale={zhCN}
                  inline
                />
              </div>

              <div className="flex justify-center gap-4">
                <button
                  onClick={closePicker}
                  className="px-6 py-2 rounded-lg border border-paper text-ink-light hover:bg-paper/50 transition-colors text-sm"
                >
                  取消
                </button>
                <button
                  onClick={confirm}
                  className="px-6 py-2 bg-crimson text-white rounded-lg hover:bg-crimson-light transition-colors text-sm"
                >
                  确定
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
}