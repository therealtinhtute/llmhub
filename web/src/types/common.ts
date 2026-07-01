/**
 * 通用类型定义
 */

export type Theme = 'light' | 'dark';

export type Language = 'en' | 'vi';

export interface ApiResponse<T = unknown> {
  data?: T;
  error?: string;
  message?: string;
}

export interface PaginationState {
  currentPage: number;
  pageSize: number;
  totalPages: number;
  totalItems?: number;
}

export interface LoadingState {
  isLoading: boolean;
  error: Error | null;
}

// 泛型异步状态
export interface AsyncState<T> extends LoadingState {
  data: T | null;
}
