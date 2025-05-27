%{
package selectorconfigparser
%}

%union {
}

%token ABS_OPERATOR
%token ANY
%token ARRAY_OPERATOR
%token BOOL_VALUE
%token COLUMN_IDENTIFIER
%token COLUMN_OPERATOR
%token COMMA
%token CONJUNCTION
%token DATE_ARGUMENT
%token DATE_OPERATOR
%token INT_VALUE
%token IS
%token LEFT_BRACKET
%token LEFT_PARENTHESIS
%token NOT
%token NULL
%token NUMBER_PART_OPERATOR
%token QUOTED_VALUE
%token RIGHT_BRACKET
%token RIGHT_PARENTHESIS
%token OPERATOR
%token UNKNOWN
%token VALUE_PLACEHOLDER

%%
clause: term
      | term CONJUNCTION clause
      | NOT clause
;

column: COLUMN_IDENTIFIER
      | COLUMN_OPERATOR LEFT_PARENTHESIS column RIGHT_PARENTHESIS
      | DATE_OPERATOR LEFT_PARENTHESIS date_operator_value COMMA column RIGHT_PARENTHESIS
      | NUMBER_PART_OPERATOR LEFT_PARENTHESIS column COMMA number_part_value RIGHT_PARENTHESIS
;

term:   column OPERATOR ANY value
      | column OPERATOR value
      | column null_check
      | LEFT_PARENTHESIS clause RIGHT_PARENTHESIS
;

value:  VALUE_PLACEHOLDER
      | BOOL_VALUE
      | INT_VALUE
      | QUOTED_VALUE
      | ARRAY_OPERATOR LEFT_BRACKET array_value RIGHT_BRACKET
      | LEFT_PARENTHESIS value RIGHT_PARENTHESIS
;

array_value: value
           | value COMMA array_value
;

date_operator_value: VALUE_PLACEHOLDER
                   | DATE_ARGUMENT
                   | LEFT_PARENTHESIS date_operator_value RIGHT_PARENTHESIS
;

number_part_value: VALUE_PLACEHOLDER
                   | INT_VALUE
                   | LEFT_PARENTHESIS number_part_value RIGHT_PARENTHESIS
;

null_check: IS NULL
      | IS NOT NULL
;

%%

