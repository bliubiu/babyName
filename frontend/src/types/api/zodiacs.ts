export interface ZodiacInfo {
  animal: string;
  // Add other zodiac properties as needed based on actual API response
  [key: string]: any;
}

export interface ZodiacsResponse {
  success: boolean;
  message?: string;
  data: ZodiacInfo[];
}