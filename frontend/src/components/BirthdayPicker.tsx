'use client';

import { useState } from 'react';
import DatePicker from 'react-datepicker';
import 'react-datepicker/dist/react-datepicker.css';
import { zhCN } from 'date-fns/locale';
import { formatLunarDate, formatSolarDate } from '@/lib/lunarUtils';

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

export function BirthdayPicker({ onConfirm, initialDisplay = '请选择出生时间' }: BirthdayPickerProps) {
  const [showModal, setShowModal] = useState(false);
  const [selectedDate, setSelectedDate] = useState<Date>(new Date());
  const [calendarType, setCalendarType] = useState<'solar' | 'lunar'>('solar');
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
        birthType: calendarType
      });
      if (calendarType === 'solar') {
        setDisplay(formatSolarDate(selectedDate));
      } else {
        setDisplay(formatLunarDate(selectedDate));
      }
    }
    closePicker();
  };

  return (
    <>
      <button
        type="button"
        onClick={openPicker}
        className="flex-1 max-w-md px-4 py-2 border border-stone-300 bg-white text-left text-red-600 hover:border-red-500 transition-colors"
      >
        {display}
      </button>

      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg w-full max-w-md mx-4 overflow-hidden">
            <div className="text-center py-4 border-b border-stone-200">
              <span className="text-lg font-medium text-stone-800">选择出生时间</span>
            </div>

            <div className="p-6">
              <div className="text-center text-lg font-medium mb-4">
                {calendarType === 'solar' ? (
                  <span>公历:{selectedDate.getFullYear()}年{selectedDate.getMonth() + 1}月{selectedDate.getDate()}日 {selectedDate.getHours()}时{selectedDate.getMinutes()}分</span>
                ) : (
                  <span>{formatLunarDate(selectedDate)}</span>
                )}
              </div>

              <div className="flex mb-6">
                <button
                  onClick={() => setCalendarType('solar')}
                  className={`flex-1 py-2 text-center font-medium ${calendarType === 'solar' ? 'bg-red-700 text-white' : 'bg-white text-red-700 border border-red-700'}`}
                  style={{ borderTopLeftRadius: '4px', borderBottomLeftRadius: '4px' }}
                >
                  公历
                </button>
                <button
                  onClick={() => setCalendarType('lunar')}
                  className={`flex-1 py-2 text-center font-medium ${calendarType === 'lunar' ? 'bg-red-700 text-white' : 'bg-white text-red-700 border border-red-700'}`}
                  style={{ borderTopRightRadius: '4px', borderBottomRightRadius: '4px', borderLeft: 'none' }}
                >
                  农历
                </button>
              </div>

              <div className="mb-6">
                <DatePicker
                  selected={selectedDate}
                  onChange={(date: Date | null) => setSelectedDate(date || new Date())}
                  showTimeSelect
                  timeFormat="HH:mm"
                  timeIntervals={15}
                  dateFormat="yyyy-MM-dd HH:mm"
                  className="w-full p-3 border border-stone-300 rounded-lg"
                  locale={zhCN}
                />
              </div>

              <div className="flex justify-center gap-4">
                <button
                  onClick={closePicker}
                  className="px-6 py-2 border border-stone-300 rounded-lg text-stone-700 hover:bg-stone-50 transition-colors"
                >
                  取消
                </button>
                <button
                  onClick={confirm}
                  className="px-6 py-2 bg-red-700 text-white rounded-lg hover:bg-red-800 transition-colors"
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
