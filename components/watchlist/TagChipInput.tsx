'use client';

import React, { useState, useRef, useMemo } from 'react';

interface TagChipInputProps {
  /** Current list of tags */
  tags: string[];
  /** Called when tags change (add or remove) */
  onChange: (tags: string[]) => void;
  /** Previously-used tags for autocomplete suggestions */
  suggestions?: string[];
  /** Placeholder text when no tags exist */
  placeholder?: string;
  /** Maximum number of tags allowed */
  maxTags?: number;
  /** Maximum length of a single tag */
  maxTagLength?: number;
}

export default function TagChipInput({
  tags,
  onChange,
  suggestions = [],
  placeholder = 'Add a tag...',
  maxTags = 50,
  maxTagLength = 100,
}: TagChipInputProps) {
  const [inputValue, setInputValue] = useState('');
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const inputRef = useRef<HTMLInputElement>(null);

  // Filter suggestions based on current input, excluding already-selected tags
  const filteredSuggestions = useMemo(() => {
    if (inputValue.trim().length === 0) return [];
    const query = inputValue.trim().toLowerCase();
    return suggestions
      .filter((s) => s.toLowerCase().includes(query) && !tags.includes(s.toLowerCase()))
      .slice(0, 6);
  }, [inputValue, suggestions, tags]);

  const addTag = (rawTag: string) => {
    const tag = rawTag.trim().toLowerCase();
    if (tag.length === 0) return;
    if (tag.length > maxTagLength) return;
    if (tags.length >= maxTags) return;
    if (tags.includes(tag)) return; // deduplicate

    onChange([...tags, tag]);
    setInputValue('');
    setShowSuggestions(false);
    setHighlightedIndex(-1);
  };

  const removeTag = (tagToRemove: string) => {
    onChange(tags.filter((t) => t !== tagToRemove));
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault();
      if (highlightedIndex >= 0 && filteredSuggestions[highlightedIndex]) {
        addTag(filteredSuggestions[highlightedIndex]);
      } else {
        addTag(inputValue);
      }
    } else if (e.key === 'Backspace' && inputValue === '' && tags.length > 0) {
      // Remove last tag
      onChange(tags.slice(0, -1));
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      setHighlightedIndex((prev) => (prev < filteredSuggestions.length - 1 ? prev + 1 : 0));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setHighlightedIndex((prev) => (prev > 0 ? prev - 1 : filteredSuggestions.length - 1));
    } else if (e.key === 'Escape') {
      setShowSuggestions(false);
      setHighlightedIndex(-1);
    }
  };

  return (
    <div className="relative">
      {/* Chip container with inline input */}
      <div
        className="flex flex-wrap items-center gap-1.5 p-2 border rounded-lg
          bg-white dark:bg-gray-900 border-gray-300 dark:border-gray-600
          focus-within:ring-2 focus-within:ring-blue-500 focus-within:border-blue-500
          transition-colors min-h-[38px]"
        onClick={() => inputRef.current?.focus()}
      >
        {tags.map((tag) => (
          <span
            key={tag}
            className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full
              text-xs font-medium bg-blue-100 text-blue-800
              dark:bg-blue-900/50 dark:text-blue-200"
          >
            {tag}
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                removeTag(tag);
              }}
              className="hover:text-blue-600 dark:hover:text-blue-100
                focus:outline-none focus:text-blue-600 ml-0.5"
              aria-label={`Remove tag: ${tag}`}
            >
              &times;
            </button>
          </span>
        ))}
        <input
          ref={inputRef}
          type="text"
          value={inputValue}
          onChange={(e) => {
            setInputValue(e.target.value);
            setShowSuggestions(true);
            setHighlightedIndex(-1);
          }}
          onKeyDown={handleKeyDown}
          onFocus={() => setShowSuggestions(true)}
          onBlur={() => {
            // Delay to allow clicking suggestions
            setTimeout(() => setShowSuggestions(false), 150);
          }}
          placeholder={tags.length === 0 ? placeholder : ''}
          className="flex-1 min-w-[80px] outline-none bg-transparent text-sm
            text-gray-900 dark:text-gray-100 placeholder-gray-400
            dark:placeholder-gray-500"
          aria-label="Add a tag. Type and press Enter to create."
          disabled={tags.length >= maxTags}
        />
      </div>

      {/* Tag suggestions dropdown */}
      {showSuggestions && filteredSuggestions.length > 0 && (
        <ul
          className="absolute z-50 mt-1 w-full bg-white dark:bg-gray-800
            border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg
            max-h-48 overflow-y-auto"
          role="listbox"
        >
          {filteredSuggestions.map((suggestion, index) => (
            <li
              key={suggestion}
              role="option"
              aria-selected={index === highlightedIndex}
              className={`px-3 py-2 text-sm cursor-pointer transition-colors
                ${
                  index === highlightedIndex
                    ? 'bg-blue-50 dark:bg-blue-900/30 text-blue-800 dark:text-blue-200'
                    : 'text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700/50'
                }`}
              onMouseDown={(e) => {
                e.preventDefault(); // Prevent blur
                addTag(suggestion);
              }}
              onMouseEnter={() => setHighlightedIndex(index)}
            >
              {suggestion}
            </li>
          ))}
        </ul>
      )}

      {/* Hint text */}
      {tags.length >= maxTags && (
        <p className="mt-1 text-xs text-gray-400">Maximum {maxTags} tags reached.</p>
      )}
    </div>
  );
}
