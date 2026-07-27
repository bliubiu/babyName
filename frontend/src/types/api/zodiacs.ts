export interface ZodiacInfo {
  id: number;
  name: string;
  wuxing: string;
  compatible: string[];
  conflicting: string[];
  avoid_chars: string;
  lucky_number: number[];
  lucky_color: string;
  lucky_direction: string;
  good_pianpang: string;
  bad_pianpang: string;
}

export interface ZodiacsResponse {
  success: boolean;
  message?: string;
  data: ZodiacInfo[];
}