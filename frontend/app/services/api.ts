// API service for connecting to backend endpoints

const API_BASE = process.env.REACT_APP_API_URL || "http://localhost:8080/api";

export interface Story {
  id: number;
  title: string;
  week_wumber: number; // Note: typo in API
  day_letter: string;
}

export interface Line {
  text: string[];
  audio_url?: string;
  has_vocab_or_grammar: boolean;
}

export interface PageData {
  story_id: string;
  story_title: string;
  lines: Line[];
}

export interface Page2Data extends PageData {
  vocab_bank: string[];
}

export interface Page3Data extends PageData {
  grammar_point: string;
}

export interface APIResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
}

interface StoriesResponse {
  stories: Story[];
}

class ApiService {
  private async fetchAPI<T>(
    endpoint: string,
    options?: RequestInit,
  ): Promise<APIResponse<T>> {
    try {
      const response = await fetch(`${API_BASE}${endpoint}`, {
        headers: {
          "Content-Type": "application/json",
          ...options?.headers,
        },
        ...options,
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      console.error("API request failed:", error);
      return {
        success: false,
        error: error instanceof Error ? error.message : "Unknown error",
      };
    }
  }

  async getStories(): Promise<APIResponse<StoriesResponse>> {
    return this.fetchAPI<StoriesResponse>("/stories");
  }

  async getStoryPage1(id: string): Promise<APIResponse<PageData>> {
    return this.fetchAPI<PageData>(`/stories/${id}/page1`);
  }

  async getStoryPage2(id: string): Promise<APIResponse<Page2Data>> {
    return this.fetchAPI<Page2Data>(`/stories/${id}/page2`);
  }

  async getStoryPage3(id: string): Promise<APIResponse<Page3Data>> {
    return this.fetchAPI<Page3Data>(`/stories/${id}/page3`);
  }

  async checkVocab(id: string, answers: any[]): Promise<APIResponse<any>> {
    return this.fetchAPI(`/stories/${id}/check-vocab`, {
      method: "POST",
      body: JSON.stringify({ answers }),
    });
  }
}

export const api = new ApiService();
